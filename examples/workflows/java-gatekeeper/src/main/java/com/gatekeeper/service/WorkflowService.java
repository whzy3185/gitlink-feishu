package com.gatekeeper.service;

import com.gatekeeper.utils.CommandLineExecutor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.util.StringUtils;

/**
 * GitLink 自动化 PR 评审工作流：通过 {@code gitlink-cli} 串联「查看 PR → 摘要 → 回写评论」。
 */
@Slf4j
@Service
public class WorkflowService {

    /** 本机 npm 全局安装的 gitlink-cli 可执行文件（Windows .cmd 脚本） */
    private static final String GITLINK_CLI =
            "C:\\Users\\jmqxx\\AppData\\Roaming\\npm\\gitlink-cli.cmd";

    /**
     * 执行完整的 PR 自动化评审工作流。
     *
     * @param owner    仓库所有者（组织或用户）
     * @param repo     仓库名称
     * @param prNumber Pull Request 编号
     * @return 步骤二生成的 Markdown 摘要报告 {@code summaryReport}
     */
    public String runPrReviewWorkflow(String owner, String repo, String prNumber) {
        validateParams(owner, repo, prNumber);
        log.info("开始执行 PR 自动化评审工作流：owner={}, repo={}, prNumber={}", owner, repo, prNumber);

        // 步骤一：获取 PR 状态（使用 --id，避免 -i 在 cmd /c 链路中被误解析）
        log.info("[步骤一] 正在获取 PR 状态…");
        String prViewOutput = viewPullRequest(owner, repo, prNumber);
        if (isCliFailure(prViewOutput)) {
            log.warn("[步骤一] PR 查询可能失败，仍将尝试后续步骤。输出：\n{}", prViewOutput);
        } else {
            log.info("[步骤一] PR 状态查询完成，输出如下：\n{}", prViewOutput);
        }

        // 步骤二：优先 workflow +pr-summary；本地 CLI 不支持时回退为 pr +view 文本摘要
        log.info("[步骤二] 正在生成 PR 摘要报告…");
        String summaryReport = generateSummaryReport(owner, repo, prNumber, prViewOutput);
        if (!StringUtils.hasText(summaryReport)) {
            log.warn("[步骤二] 摘要报告为空，后续仍将尝试以评论形式回写");
            summaryReport = buildFallbackSummary(prViewOutput);
        } else {
            log.info("[步骤二] 摘要报告已就绪，字符数={}", summaryReport.length());
            if (log.isDebugEnabled()) {
                log.debug("[步骤二] 摘要报告内容：\n{}", summaryReport);
            }
        }

        // 步骤三：本地 CLI 无 pr +review，使用 pr +comment 将摘要回写为 PR 评论
        log.info("[步骤三] 正在通过 pr +comment 回写评审摘要…");
        String commentOutput = postPullRequestComment(owner, repo, prNumber, summaryReport);
        log.info("[步骤三] 评论已提交，CLI 输出如下：\n{}", commentOutput);
        log.info("PR 自动化评审工作流执行完毕：owner={}, repo={}, prNumber={}", owner, repo, prNumber);

        return summaryReport;
    }

    /** 调用 {@code pr +view}，PR 编号使用官方长参数 {@code --id}（等价于 --number）。 */
    private String viewPullRequest(String owner, String repo, String prNumber) {
        return CommandLineExecutor.run(
                GITLINK_CLI, "pr", "+view",
                "--owner", owner,
                "--repo", repo,
                "--id", prNumber);
    }

    /**
     * 生成摘要：先尝试 {@code workflow +pr-summary}；失败或不可用时用步骤一的 PR 详情构造 Markdown 回退摘要。
     */
    private String generateSummaryReport(
            String owner, String repo, String prNumber, String prViewOutput) {
        try {
            log.info("[步骤二] 尝试官方命令 workflow +pr-summary …");
            String aiSummary = CommandLineExecutor.run(
                    GITLINK_CLI, "workflow", "+pr-summary",
                    "--owner", owner,
                    "--repo", repo,
                    "--number", prNumber,
                    "--format", "markdown");
            if (!isCliFailure(aiSummary) && StringUtils.hasText(aiSummary)) {
                log.info("[步骤二] 已使用 workflow +pr-summary 生成摘要");
                return aiSummary;
            }
            log.warn("[步骤二] workflow +pr-summary 无有效输出，启用 pr +view 回退。原因片段：{}",
                    truncateForLog(aiSummary, 200));
        } catch (Exception e) {
            log.warn("[步骤二] workflow +pr-summary 调用异常，启用 pr +view 回退：{}", e.getMessage());
        }
        return buildFallbackSummary(prViewOutput);
    }

    /** 使用 {@code pr +comment} 将摘要作为评论发表（当前 CLI 版本不支持 pr +review）。 */
    private String postPullRequestComment(
            String owner, String repo, String prNumber, String summaryReport) {
        String body = StringUtils.hasText(summaryReport)
                ? summaryReport
                : buildFallbackSummary("");
        return CommandLineExecutor.run(
                GITLINK_CLI, "pr", "+comment",
                "--owner", owner,
                "--repo", repo,
                "--id", prNumber,
                "--body", body);
    }

    private static String buildFallbackSummary(String prViewOutput) {
        StringBuilder builder = new StringBuilder();
        builder.append("## PR 自动化摘要（本地回退）").append(System.lineSeparator()).append(System.lineSeparator());
        if (StringUtils.hasText(prViewOutput)) {
            builder.append(prViewOutput.trim());
        } else {
            builder.append("_未能获取 PR 详情，请检查 owner/repo/PR 编号及 gitlink-cli 登录状态。_");
        }
        return builder.toString();
    }

    /** 根据 CLI 合并输出判断命令是否执行失败（不抛异常，仅用于回退逻辑）。 */
    private static boolean isCliFailure(String output) {
        if (!StringUtils.hasText(output)) {
            return true;
        }
        String normalized = output.toLowerCase();
        return normalized.contains("unknown command")
                || normalized.contains("unknown shorthand flag")
                || normalized.contains("unknown flag")
                || normalized.contains("error:")
                || normalized.contains("failed");
    }

    private static String truncateForLog(String text, int maxLen) {
        if (text == null) {
            return "";
        }
        String trimmed = text.replace('\n', ' ').trim();
        return trimmed.length() <= maxLen ? trimmed : trimmed.substring(0, maxLen) + "…";
    }

    private static void validateParams(String owner, String repo, String prNumber) {
        if (!StringUtils.hasText(owner)) {
            throw new IllegalArgumentException("owner 不能为空");
        }
        if (!StringUtils.hasText(repo)) {
            throw new IllegalArgumentException("repo 不能为空");
        }
        if (!StringUtils.hasText(prNumber)) {
            throw new IllegalArgumentException("prNumber 不能为空");
        }
    }
}
