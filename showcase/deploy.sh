# 展示页部署命令（在本地终端手动执行）

# 1. 用 SCP 上传展示页到 ECS（需要手动输入密码：pd1@YwC#WRFVHkXc8nvu!4）
scp "d:/自用/self/word/大三下/软件演化/gitlink-cli/showcase/index.html" root@121.41.210.165:/opt/showcase/index.html

# 2. SSH 到 ECS（密码：pd1@YwC#WRFVHkXc8nvu!4）
ssh root@121.41.210.165

# 登录后在 ECS 上执行：
mkdir -p /opt/showcase
docker run -d --name showcase -p 8080:80 -v /opt/showcase:/usr/share/nginx/html:ro --restart unless-stopped nginx:alpine

# 3. 访问
# 浏览器打开 http://121.41.210.165:8080
