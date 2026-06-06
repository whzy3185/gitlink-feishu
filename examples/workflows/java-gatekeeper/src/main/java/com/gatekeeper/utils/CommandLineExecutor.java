package com.gatekeeper.utils;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.nio.charset.Charset;
import java.nio.charset.UnsupportedCharsetException;
import java.util.Arrays;
import java.util.stream.Collectors;

/**
 * 命令行执行工具：通过 {@link ProcessBuilder} 在子进程中静默运行外部命令，
 * 并合并捕获标准输出与错误输出。
 */
public final class CommandLineExecutor {

    /**
     * 子进程输出编码：Windows 控制台多为 GBK，避免中文乱码；其他系统使用 JVM 默认编码。
     */
    private static final Charset OUTPUT_CHARSET = resolveOutputCharset();

    private CommandLineExecutor() {
    }

    /**
     * 在系统后台静默执行一条外部命令，将标准输出与错误输出合并为一段文本返回。
     *
     * <p>示例：{@code run("C:\\path\\to\\gitlink-cli.cmd", "user", "+me")}</p>
     * <p>Windows 下若首参数为 {@code .cmd}/{@code .bat}，将自动通过 {@code cmd.exe /c} 启动。</p>
     *
     * @param command 可执行文件路径或名称，以及后续参数（不可为空）
     * @return 子进程的标准输出与错误输出的完整文本（无输出时返回空字符串）
     * @throws IllegalArgumentException 未传入任何命令片段时
     * @throws CommandExecutionException 进程无法启动、被中断或等待结束时发生 I/O 错误时
     */
    public static String run(String... command) {
        if (command == null || command.length == 0) {
            throw new IllegalArgumentException("命令参数不能为空");
        }
        for (int i = 0; i < command.length; i++) {
            if (command[i] == null || command[i].isBlank()) {
                throw new IllegalArgumentException("命令参数第 " + (i + 1) + " 段不能为空");
            }
        }

        String[] processCommand = normalizeCommandForWindows(command);
        String commandLine = Arrays.stream(processCommand).collect(Collectors.joining(" "));
        ProcessBuilder processBuilder = new ProcessBuilder(processCommand);
        // 清空代理环境变量，避免子进程继承终端代理导致连接超时
        processBuilder.environment().put("http_proxy", "");
        processBuilder.environment().put("https_proxy", "");
        processBuilder.environment().put("all_proxy", "");
        // 不向控制台继承标准流，在后台以管道方式静默执行
        processBuilder.redirectErrorStream(true);

        Process process;
        try {
            process = processBuilder.start();
        } catch (IOException e) {
            throw new CommandExecutionException(
                    "无法启动命令进程：" + commandLine, commandLine, e);
        }

        try {
            String output = readAllText(process.getInputStream());
            process.waitFor();
            // 无论退出码是否为 0，均返回已捕获的标准输出与错误输出合并文本
            return output;
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            process.destroyForcibly();
            throw new CommandExecutionException(
                    "等待命令执行结果被中断：" + commandLine, commandLine, e);
        } catch (IOException e) {
            process.destroyForcibly();
            throw new CommandExecutionException(
                    "读取命令输出时发生 I/O 错误：" + commandLine, commandLine, e);
        } finally {
            if (process.isAlive()) {
                process.destroyForcibly();
            }
        }
    }

    private static Charset resolveOutputCharset() {
        if (isWindows()) {
            try {
                return Charset.forName("GBK");
            } catch (UnsupportedCharsetException ignored) {
                return Charset.defaultCharset();
            }
        }
        return Charset.defaultCharset();
    }

    /**
     * Windows 下通过 {@code cmd.exe /c} 执行批处理脚本，避免 ProcessBuilder 无法直接启动 {@code .cmd}/{@code .bat}。
     */
    private static String[] normalizeCommandForWindows(String... command) {
        if (!isWindows()) {
            return command;
        }
        String executable = command[0].toLowerCase();
        if (!executable.endsWith(".cmd") && !executable.endsWith(".bat")) {
            return command;
        }
        String[] wrapped = new String[command.length + 2];
        wrapped[0] = resolveWindowsCmdExe();
        wrapped[1] = "/c";
        System.arraycopy(command, 0, wrapped, 2, command.length);
        return wrapped;
    }

    private static boolean isWindows() {
        String osName = System.getProperty("os.name", "");
        return osName.toLowerCase().contains("win");
    }

    /** 优先使用系统 ComSpec，回退到默认 cmd.exe 绝对路径 */
    private static String resolveWindowsCmdExe() {
        String comSpec = System.getenv("ComSpec");
        if (comSpec != null && !comSpec.isBlank()) {
            return comSpec;
        }
        return "C:\\Windows\\System32\\cmd.exe";
    }

    /**
     * 从子进程输入流中读取全部文本（已合并 stderr 时即为标准输出 + 错误输出）。
     */
    private static String readAllText(InputStream inputStream) throws IOException {
        StringBuilder builder = new StringBuilder();
        try (BufferedReader reader = new BufferedReader(
                new InputStreamReader(inputStream, OUTPUT_CHARSET))) {
            String line;
            while ((line = reader.readLine()) != null) {
                if (!builder.isEmpty()) {
                    builder.append(System.lineSeparator());
                }
                builder.append(line);
            }
        }
        return builder.toString();
    }

    /**
     * 命令执行失败时抛出的运行时异常，携带便于排查的完整命令行信息。
     */
    public static class CommandExecutionException extends RuntimeException {

        private final String commandLine;

        public CommandExecutionException(
                String message, String commandLine, Throwable cause) {
            super(message, cause);
            this.commandLine = commandLine;
        }

        /** 实际执行的完整命令行（参数以空格拼接，便于日志排查） */
        public String getCommandLine() {
            return commandLine;
        }
    }
}
