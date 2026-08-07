package feishu

import (
	"context"
	"fmt"
	"strings"
	"time"

	prshortcut "github.com/gitlink-org/gitlink-cli/shortcuts/pr"
	"github.com/gitlink-org/gitlink-cli/shortcuts/workflow"
)

func (e *ReviewGatewayExecutor) prepareCommonReview(ctx context.Context, job ReviewGatewayJob, result ReviewGatewayExecutionResult, now time.Time) (ReviewGatewayExecutionResult, error) {
	if e.ActionPlans == nil {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("review action plan runtime and store are required"))
	}
	runtime, err := e.runtimeForJob(job)
	if err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	identity, ok := findReviewIdentity(e.IdentityBindings, job.RequestedBy, job.InstallationID)
	if !ok {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("Feishu account is not bound to a GitLink account"))
	}
	owner, repo, err := splitReviewGatewayRepository(job.Repository)
	if err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	reviewContext, err := workflow.FetchReviewContext(runtime, workflow.ReviewContextOptions{
		Owner: owner, Repo: repo, Number: job.PRNumber, VersionLimit: 100, ThreadLimit: 100,
		IncludePR: true, IncludeFiles: true, IncludeVersions: true, IncludeReviews: true, IncludeThreads: true,
	})
	if err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	populateReviewGatewayContextResult(&result, reviewContext)
	if reviewContext.Partial || reviewContext.CollectionStatus != "complete" || strings.TrimSpace(reviewContext.CurrentHeadSHA) == "" {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("complete Review context and current head are required before write planning"))
	}
	draft := buildReviewDraftPreview(reviewContext)
	content := strings.TrimSpace(job.Argument)
	if content == "" {
		content = strings.TrimSpace(strings.Join([]string{
			draft.Summary,
			"Current decision: " + firstNonEmpty(draft.Decision, "pending"),
			"Suggested next step: " + firstNonEmpty(draft.NextStep, "owner review"),
			"Source head: " + reviewContext.CurrentHeadSHA,
		}, "\n"))
	}
	plan := NewReviewActionPlan(job, identity.GitLinkLogin, reviewContext.CurrentHeadSHA, reviewContext.WorkItem.SourceFingerprint, content, now)
	plan, err = e.ActionPlans.CreateReviewActionPlan(ctx, plan)
	if err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	result.ActionPlan, result.Draft = &plan, &draft
	result.ResultCard = buildReviewGatewayResultCard(job, result, nil)
	result.ReadOnlyGitLink, result.MutatesGitLink = true, false
	result.Message = fmt.Sprintf("common Review ActionPlan %s is ready for local confirmation; Ref: %s", plan.PlanID, plan.RequestID)
	return result, nil
}

func (e *ReviewGatewayExecutor) confirmCommonReview(ctx context.Context, job ReviewGatewayJob, result ReviewGatewayExecutionResult, now time.Time) (ReviewGatewayExecutionResult, error) {
	if e.ActionPlans == nil {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("review action plan runtime and store are required"))
	}
	plan, err := e.ActionPlans.GetReviewActionPlan(ctx, job.Argument)
	if err != nil {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("load review action plan: %w", err))
	}
	result.ActionPlan, result.Repository, result.PRNumber = &plan, plan.Repository, plan.PRNumber
	result.ReadOnlyGitLink, result.MutatesGitLink = !e.EnableGitLinkWrite, false
	if plan.ActorID != job.RequestedBy {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("review action plan belongs to another Feishu user"))
	}
	if plan.ReviewStatus != "common" {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("only common Review writes are enabled"))
	}
	if err := e.validateReviewActionPlanScope(job, plan); err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	identity, ok := findReviewIdentity(e.IdentityBindings, job.RequestedBy, job.InstallationID)
	if !ok || identity.GitLinkLogin != plan.GitLinkLogin {
		return reviewGatewayExecutionFailure(result, fmt.Errorf("GitLink identity binding changed; create a new action plan"))
	}
	if !e.EnableGitLinkWrite {
		result.WriteResult = &ReviewWriteResult{PlanID: plan.PlanID, Status: "write_disabled", Repository: plan.Repository,
			PRNumber: plan.PRNumber, HeadSHA: plan.ExpectedHeadSHA, ReviewStatus: "common", MutationStatus: reviewMutationNone,
			RequestID: plan.RequestID}
		result.Message = "ActionPlan is valid, but GitLink writes are disabled"
		return result, nil
	}
	runtime, err := e.runtimeForJob(job)
	if err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	leaseOwner := "review-write:" + job.JobID
	plan, err = e.ActionPlans.ClaimReviewActionPlan(ctx, ReviewActionPlanClaimOptions{PlanID: plan.PlanID, ActorID: job.RequestedBy,
		LeaseOwner: leaseOwner, Now: now, LeaseDuration: 2 * time.Minute})
	if err != nil {
		return reviewGatewayExecutionFailure(result, err)
	}
	result.ActionPlan = &plan
	owner, repo, err := splitReviewGatewayRepository(plan.Repository)
	if err != nil {
		_ = e.ActionPlans.FinishReviewActionPlan(ctx, plan.PlanID, "failed", "", err.Error(), reviewMutationNone, now)
		return reviewGatewayExecutionFailure(result, err)
	}
	reviewContext, err := workflow.FetchReviewContext(runtime, workflow.ReviewContextOptions{Owner: owner, Repo: repo,
		Number: plan.PRNumber, VersionLimit: 100, ThreadLimit: 100, IncludePR: true, IncludeFiles: true,
		IncludeVersions: true, IncludeReviews: true, IncludeThreads: true})
	if err != nil {
		_ = e.ActionPlans.FinishReviewActionPlan(ctx, plan.PlanID, "failed", "", err.Error(), reviewMutationNone, now)
		return reviewGatewayExecutionFailure(result, err)
	}
	populateReviewGatewayContextResult(&result, reviewContext)
	if reviewContext.Partial || reviewContext.CollectionStatus != "complete" || reviewContext.CurrentHeadSHA != plan.ExpectedHeadSHA ||
		reviewContext.WorkItem.SourceFingerprint != plan.SourceFingerprint {
		_ = e.ActionPlans.FinishReviewActionPlan(ctx, plan.PlanID, "stale", "", "PR context changed", reviewMutationNone, now)
		result.WriteResult = &ReviewWriteResult{PlanID: plan.PlanID, Status: "stale", Outcome: "stale", Repository: plan.Repository,
			PRNumber: plan.PRNumber, HeadSHA: reviewContext.CurrentHeadSHA, ReviewStatus: "common", MutationStatus: reviewMutationNone,
			RequestID: plan.RequestID}
		result.Message = "PR context changed; ActionPlan is stale and GitLink writes remain zero"
		return result, nil
	}
	write := prshortcut.ExecuteCommonReview(runtime, prshortcut.CommonReviewOptions{Owner: owner, Repository: repo,
		PRNumber: plan.PRNumber, Content: plan.Content, ExpectedHead: plan.ExpectedHeadSHA, ExpectedActor: plan.GitLinkLogin,
		RequestID: plan.RequestID, BeforePOST: func() error {
			return e.ActionPlans.MarkReviewActionPlanWriteStarted(ctx, plan.PlanID, leaseOwner, now)
		}})
	return e.finishCommonReviewWrite(ctx, result, plan, write, now)
}

func (e *ReviewGatewayExecutor) finishCommonReviewWrite(ctx context.Context, result ReviewGatewayExecutionResult, plan ReviewActionPlan, write prshortcut.CommonReviewResult, now time.Time) (ReviewGatewayExecutionResult, error) {
	terminal, status, mutation := "failed", write.Status, reviewMutationNone
	switch write.Status {
	case "verified", "duplicate":
		terminal, status = "completed", "completed"
	case "stale":
		terminal = "stale"
	case "unknown":
		terminal, status, mutation = "unknown", "unknown_needs_reconciliation", reviewMutationPossible
	}
	if write.Mutated {
		mutation = reviewMutationConfirmed
	}
	if err := e.ActionPlans.FinishReviewActionPlan(ctx, plan.PlanID, terminal, write.ReviewID, write.Error, mutation, now); err != nil {
		_ = e.ActionPlans.MarkReviewActionPlanUnknown(ctx, plan.PlanID, "GitLink result could not be persisted", mutation, now)
		status = "unknown_needs_reconciliation"
	}
	result.ReadOnlyGitLink, result.MutatesGitLink = write.POSTCount == 0, write.Mutated
	result.WriteResult = &ReviewWriteResult{PlanID: plan.PlanID, Status: status, Outcome: write.Status, ReviewID: write.ReviewID,
		Repository: plan.Repository, PRNumber: plan.PRNumber, HeadSHA: write.CurrentHead, ReviewStatus: "common",
		MutationStatus: mutation, Mutated: write.Mutated, RequestID: plan.RequestID}
	result.Warnings = append(result.Warnings, write.Warnings...)
	if status == "unknown_needs_reconciliation" {
		result.WriteResult.Reconciliation = "verify the Review by request ID before any retry"
	} else if write.Status == "verified" {
		result.WriteResult.Reconciliation = "verified by GitLink GET read-back"
	}
	result.Message = fmt.Sprintf("common Review %s; Ref: %s", write.Status, plan.RequestID)
	if latest, err := e.ActionPlans.GetReviewActionPlan(ctx, plan.PlanID); err == nil {
		result.ActionPlan = &latest
	}
	return result, nil
}

func (e *ReviewGatewayExecutor) validateReviewActionPlanScope(job ReviewGatewayJob, plan ReviewActionPlan) error {
	if strings.TrimSpace(plan.InstallationID) == "" || strings.TrimSpace(plan.SourceChatID) == "" || strings.TrimSpace(plan.Repository) == "" {
		return fmt.Errorf("review action plan is missing its original installation, chat, or repository scope")
	}
	if job.InstallationID != plan.InstallationID {
		return fmt.Errorf("review action plan belongs to another GitLink installation")
	}
	if job.ChatID != plan.SourceChatID {
		return fmt.Errorf("review action plan must be confirmed in its source chat")
	}
	installation, exists := e.Installations[plan.InstallationID]
	if !exists || !installation.Enabled || installation.OperationMode != "write" {
		return fmt.Errorf("review action plan GitLink installation is unavailable for writes")
	}
	if !containsReviewGatewayString(installation.AllowedRepositories, plan.Repository) || !containsReviewGatewayString(job.Repositories, plan.Repository) {
		return fmt.Errorf("review action plan repository is outside its original scope")
	}
	return nil
}
