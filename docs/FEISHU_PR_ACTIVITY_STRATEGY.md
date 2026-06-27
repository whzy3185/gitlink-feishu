# PR Activity, Review Attribution, and Snapshot Strategy

Status: next-stage design. No GitLink write operation is implemented here.

## Goal

Provide repository owners with a general, cross-repository view of:

- all current open pull requests;
- merged and closed/rejected totals;
- state transitions since the previous assessment;
- whether a pull request received a formal review;
- whether conversation feedback came from the submitter, a reviewer, a
  maintainer, another participant, a bot, or a system event;
- whether review/comment content changed since the previous snapshot.

The model must not depend on a specific repository, user login, PR number, or
organization role name.

## Verified Read Sources

The GitLink API surfaces needed by this design are read-only:

```text
GET /v1/{owner}/{repo}/pulls
GET /v1/{owner}/{repo}/pulls/{number}
GET /v1/{owner}/{repo}/pulls/{number}/reviews
GET /v1/{owner}/{repo}/issues/{issue_id}/journals
GET repository members when the current identity has permission
```

Local validation confirmed:

- list responses expose PR state, author, associated issue ID, and pagination
  totals;
- formal reviews expose reviewer identity, status, content, and time;
- journals expose actor identity, comment text, state events, created time, and
  updated time;
- repository-member lookup may return 401 for an unauthenticated read. The
  implementation must degrade to `participant`, not guess maintainer status.

## Actor Classification

Normalize identities by stable user ID first and login second.

| Actor class | Evidence |
| --- | --- |
| `submitter` | Actor matches the PR author |
| `reviewer` | Actor owns a formal review or is in the assigned reviewer set |
| `maintainer` | Repository membership data proves a configured privileged role |
| `participant` | Authenticated human who is none of the above |
| `bot` | Explicit bot/application identity |
| `system` | State transition or generated event without human review content |
| `unknown` | Identity is incomplete |

Role precedence:

```text
system/bot -> submitter -> formal reviewer -> maintainer -> participant -> unknown
```

A user can be both maintainer and reviewer. Event attribution records the most
specific event relationship (`reviewer`) and may retain `is_maintainer=true` as
an additional property.

## Review Standard

Do not treat every comment as a review.

### Authoritative review

A formal review object with:

```text
status: approved | rejected | common
reviewer identity
created_at
```

is authoritative review evidence.

### Review-like journal feedback

A journal comment is review feedback only when:

1. it has non-empty human-authored content;
2. it is not a creation/status/system event;
3. the actor is a formal/assigned reviewer or a proven maintainer;
4. the actor is not the PR submitter, unless the UI explicitly marks a
   self-review;
5. the normalized content is not only an acknowledgement such as `LGTM`,
   `thanks`, or a generated status line, unless the product policy explicitly
   enables acknowledgement reviews.

When member data is unavailable, a comment from an unassigned actor remains
`participant_feedback`, not `maintainer_review`.

## Risk Is Separate From Review

Current `workflow +repo-report` risk is rule-based. A list-metadata keyword hit
is a risk hint, not proof that a reviewer found a problem.

The next-stage output should keep separate fields:

```text
metadata_risk_hint
code_change_risk
formal_review_status
review_feedback_status
merge_readiness
```

Detailed code risk requires files and commits. Bulk list metadata alone must
not be presented as a formal review conclusion.

## Snapshot Model

Recommended local snapshot:

```json
{
  "schema_version": 1,
  "repository": "owner/repo",
  "generated_at": "RFC3339",
  "totals": {
    "open": 0,
    "merged": 0,
    "closed": 0
  },
  "pull_requests": [
    {
      "number": 1,
      "state": "open",
      "author_id": "stable-id",
      "updated_at": "RFC3339",
      "head_revision": "optional",
      "formal_review_status": "unreviewed",
      "review_fingerprint": "sha256",
      "conversation_fingerprint": "sha256",
      "events": []
    }
  ]
}
```

Raw access tokens and private profile fields must never enter snapshots.

## Content Fingerprints

Normalize review/comment content before hashing:

1. normalize line endings;
2. trim leading/trailing whitespace;
3. collapse repeated whitespace outside code blocks;
4. remove generated status-only markup;
5. preserve code and semantic text;
6. hash actor ID, event type, normalized content, and event time.

Store hashes and bounded summaries by default. Raw comment content should be
included only in an explicitly local evidence file.

## Snapshot Diff

Compare the current snapshot with `--previous-snapshot` and emit:

```text
new_open_prs
newly_merged_prs
newly_closed_prs
reopened_prs
new_formal_reviews
review_status_changes
new_reviewer_feedback
edited_reviewer_feedback
submitter_responses
participant_feedback
```

State transitions are determined by PR number plus previous/current state, not
by subtracting aggregate totals.

## Fetch Strategy

Default inventory:

1. paginate all PR list states;
2. record exact list totals and basic identity/state fields;
3. compare with the previous snapshot;
4. deep-fetch reviews and journals only for new or updated PRs.

Optional full audit:

```text
--full-review-audit
```

This explicitly deep-fetches every PR and may require hundreds of API calls.
Use bounded concurrency, retry/backoff, and a request summary. It must remain
read-only.

## Planned Commands

```text
gitlink-cli workflow +pr-activity-snapshot
gitlink-cli workflow +pr-activity-diff
gitlink-cli feishu +owner-activity-digest
```

The Feishu digest should show aggregate transitions and the most important
changed PRs. It must link to GitLink for full comments rather than copying an
unbounded conversation into a card.

## Current Boundary

Implemented in the current branch:

- correct GitLink Issue and PR list filters;
- complete pagination for `workflow +repo-report` by default;
- PR lifecycle totals for open, merged, and closed/rejected states;
- explicit analyzed-count labels and scope notes;
- optional read-only formal review and journal actor attribution through
  `workflow +repo-report --include-pr-review-audit`.

The implemented audit follows the conservative review standard in this
document:

- a formal `/pulls/{number}/reviews` object marks the PR as reviewed;
- journal comments from the PR submitter are counted as submitter responses,
  not reviews;
- journal comments from an actor who also has a formal review on the PR are
  counted as reviewer feedback;
- other human comments are counted as participant feedback;
- status changes and empty/generated events are counted as system events;
- a reviewed PR is marked `needs_re_review` when a later submitter comment,
  later commit, or later PR update timestamp is newer than the latest reviewer
  feedback timestamp;
- maintainer classification is not guessed when repository-member data is not
  available.

Not implemented in the current branch:

- snapshot persistence;
- member-role enrichment;
- review-content diffing;
- Feishu activity-diff cards;
- any GitLink write operation.
