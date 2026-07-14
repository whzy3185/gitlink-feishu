package com.gatekeeper;

import com.gatekeeper.service.WorkflowService;
import com.gatekeeper.utils.CommandLineExecutor.CommandExecutionException;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.CommandLineRunner;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

@SpringBootApplication
public class GatekeeperApplication implements CommandLineRunner {

    private static final String SEPARATOR = "============================================================";

    @Autowired
    private WorkflowService workflowService;

    public static void main(String[] args) {
        SpringApplication.run(GatekeeperApplication.class, args);
    }

    @Override
    public void run(String... args) {
        System.out.println(SEPARATOR);
        System.out.println("  GitLink Gatekeeper · PR 自动化评审工作流");
        System.out.println("  测试目标：Gitlink / forgeplus · PR #42");
        System.out.println(SEPARATOR);

        try {
            String summaryReport = workflowService.runPrReviewWorkflow(
                    "Gitlink", "forgeplus", "42");

            System.out.println();
            System.out.println("【工作流摘要预览】");
            System.out.println(summaryReport);
            System.out.println();
            System.out.println(SEPARATOR);
            System.out.println("  工作流执行成功，已结束");
            System.out.println(SEPARATOR);
        } catch (CommandExecutionException e) {
            System.err.println();
            System.err.println("【命令执行失败】" + e.getMessage());
            if (e.getCommandLine() != null) {
                System.err.println("命令：" + e.getCommandLine());
            }
            e.printStackTrace();
            printFailureFooter();
        } catch (IllegalArgumentException e) {
            System.err.println();
            System.err.println("【参数错误】" + e.getMessage());
            printFailureFooter();
        } catch (Exception e) {
            System.err.println();
            System.err.println("【工作流执行异常】" + e.getMessage());
            e.printStackTrace();
            printFailureFooter();
        }
    }

    private static void printFailureFooter() {
        System.err.println();
        System.err.println(SEPARATOR);
        System.err.println("  工作流执行失败，已结束");
        System.err.println(SEPARATOR);
    }
}
