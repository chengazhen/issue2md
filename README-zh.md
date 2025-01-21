# issue2md

一个命令行和网页工具，用于将 GitHub issue 转换为 Markdown 格式文件。

>此仓库中的大部分内容是由人工智能生成的！

## 命令行模式

### 安装 issue2md 命令行工具

```bash
$ go install github.com/bigwhite/issue2md/cmd/issue2md@latest
```

### 将 Issues 转换为 Markdown

该工具支持三种操作模式：

1. **单个 Issue 下载**
```bash
issue2md <issue-url> [output-dir]
```
示例：
```bash
issue2md https://github.com/owner/repo/issues/123 downloads
```

2. **从文件批量下载**
```bash
issue2md -f <urls-file> [output-dir]
```
示例：
```bash
issue2md -f issues.txt downloads
```
`urls-file` 文件中每行包含一个 issue URL。以 # 开头的行会被忽略。

3. **下载仓库的所有 Issues**
```bash
issue2md -r <repo-url> [output-dir]
```
示例：
```bash
issue2md -r https://github.com/owner/repo downloads
```

### 重要说明

1. **输出目录**
   - 默认情况下，文件保存在 `downloads` 目录中
   - 你可以在命令最后指定其他目录

2. **GitHub API 访问限制**
   - 未认证用户：每小时 60 个请求
   - 认证用户：每小时 5,000 个请求
   - 每个 issue 需要 2 个 API 请求（一个获取 issue 详情，一个获取评论）

3. **GitHub Token**
   - 要避免访问限制，请设置你的 GitHub token：
     ```bash
     # Windows
     set GITHUB_TOKEN=your_token
     
     # Linux/Mac
     export GITHUB_TOKEN=your_token
     ```
   - 你可以在这里创建 token：https://github.com/settings/tokens
   - 如果没有设置 token，批量下载将限制为 30 个 issues

4. **功能特点**
   - 自动跳过已存在的文件
   - 显示进度和统计信息
   - 请求之间随机延迟，避免触发限制
   - 支持断点续传（中断后可继续下载）

## 网页模式

### 安装并运行 issue2md web

#### 使用 Docker 运行（推荐）

```bash
$ docker run -d -p 8080:8080 bigwhite/issue2mdweb
```

#### 从源码构建

```bash
$ git clone https://github.com/bigwhite/issue2md.git
$ make web
$ ./issue2mdweb
服务器运行在 http://0.0.0.0:8080
```

### 将 Issues 转换为 Markdown

在浏览器中打开 localhost:8080：

![](./screen-snapshot.png)

输入你想要转换的 issue URL，然后点击"Convert"按钮！
