---
title: 通过Github Action部署hexo博客到pages
date: 2021-10-04 14:41:24
tags: Hexo
---

Hexo部署到 Github Page 的一般分为下面三步

1. 写 Markdown 
2. 将 Markdown 渲染成 HTML 静态页面
3. 将渲染得到的文件提交到 `<username>.github.io` 仓库

如果以写博客为目的话，上述2，3两步其实是“写博客“之外的操作，比较影响写博客的沉浸式体验。这里可以引入现成的 Github Action ，将上述的 workflow 进行自动化。



## 准备

1. Github Pages 仓库 `<username>.github.io`

2. Blog 的源代码仓库

3. ssh-keygen 生成一对非对称密钥，公钥作为 page 仓库的 `Deploy Key`，私钥填入源代码仓库的 Secrets，供后续 Github Action 提交代码到 page 仓库

4. 安装 hexo git 部署插件 hexo-deployer-git

   ```
   npm install hexo-deployer-git --save
   ```

   

## 配置Hexo Deploy

在hexo源码工程下的 `_config.yml` 中配置下列信息

```yaml
deploy:
  type: git
  repo: git@github.com:username/<username>.github.io.git
  branch: 分支名
  message: 提交信息
```

 在执行 `hexo deploy` 命令时，hexo-deployer-git 插件大致会执行如下流程：

1. 将源代码仓库渲染成静态网页到 `.deploy_git` 目录下
2. 然后将 `.deploy_git` 目录 force push 到配置中指定的 repo 下的 branch 上，同时 git 提交信息就是配置中指定的 message

更多信息：

* [hexo部署](https://hexo.io/docs/one-command-deployment)

* [hexo-deployer-git插件](https://github.com/hexojs/hexo-deployer-git)



## 编写Github Action Workflow

整个 workflow 拆分成下面几步：

1. checkout 到源码主分支
2. 配置 node 环境
3. 配置 git 环境
4. deploy hexo 带page仓库



workflow文件如下：

```yaml
name: auto deploy

on:
  push:
    branches:
    - main
  workflow_dispatch:

jobs:

  build:
  
    runs-on: ubuntu-latest

    steps:

      - name: Checkout to main
        with:
          submodules: recursive
          fetch-depth: 0
        uses: actions/checkout@v2

      - name: Setup Node
        uses: actions/setup-node@v1
        with:
          node-version: '14'
          
      - name: Setup Git
        env:
          DEPLOY_SECRET: ${{ secrets.DEPLOY_SECRET }}
        run: |
          mkdir -p ~/.ssh
          echo "$DEPLOY_SECRET" > ~/.ssh/id_rsa
          chmod 400 ~/.ssh/id_rsa
          ssh-keyscan github.com >> ~/.ssh/known_hosts
          git config --global user.email "ngtercet@protonmail.com"
          git config --global user.name "Jaron"
        
      - name: Deploy
        run: |
          chmod +x deploy.sh
          ./deploy.sh
```



deploy 脚本如下：

```bash
#!/usr/bin/env bash

# install dependencies
npm install hexo-cli -g
npm install

# fetch mtime from git log
git ls-files -z source/_posts |
    while read -d '' path; do
    if [[ $path == *.md ]]; then
        mtime=$(git log -1 --format="@%ct" $path)
        touch -d $mtime $path
        echo "change $path mtime to $mtime"
    fi
    done

# hexo deploy
hexo clean
hexo deploy
```



## 踩坑点

#### Post 的“更新时间”不对

##### 问题

最初 Github Action 跑起来后发现博客里所有 Post 的”上次更新时间（updated属性）“都被刷新成部署的时间点了。

翻了下 Hexo 相关文档发现，Post 的 updated 属性在缺省的情况下默认是 fallback 成 mtime，也就是文件的 `Modified time`。

在 action 每次跑的时候，其实都是在构建机器先 clone 了一份源码库，所有代码文件对于构建机器来说都是新创建的文件。

而 linux 下新创建文件的 `Modified time` 就是被初始化为创建时间的：

```shell
➜  ~ date && touch test && stat test   
Mon Oct  4 03:36:08 PM CST 2021
  File: test
  Size: 0         	Blocks: 0          IO Block: 4096   regular empty file
Device: 8,7	Inode: 135137      Links: 1
Access: (0644/-rw-r--r--)  Uid: ( 1000/   jaron)   Gid: ( 1000/   jaron)
Access: 2021-10-04 15:36:08.763454559 +0800
Modify: 2021-10-04 15:36:08.763454559 +0800
Change: 2021-10-04 15:36:08.763454559 +0800
 Birth: 2021-10-04 15:36:08.763454559 +0800
```

所以上述问题也就不言而喻了。

##### 解决

1. Hexo文档的做法

   其实 Hexo 文档对这个问题也有相关描述，其中说的是 git 工作流下可以将 updated 属性 fallback 成 date，即文件创建时间。这种方式基本相当于因噎废食，直接将 updated 属性本身的语义给屏蔽了。

   > **updated_option**
   >
   > `updated_option` 控制了当 Front Matter 中没有指定 `updated` 时，`updated` 如何取值：
   >
   > - `mtime`: 使用文件的最后修改时间。这是从 Hexo 3.0.0 开始的默认行为。
   > - `date`: 使用 `date` 作为 `updated` 的值。可被用于 Git 工作流之中，因为使用 Git 管理站点时，文件的最后修改日期常常会发生改变
   > - `empty`: 直接删除 `updated`。使用这一选项可能会导致大部分主题和插件无法正常工作。

2. 根据 git log 获取 post 的 last commit time

   在 workflow 中添加一步：将 post 文件上次 git commit 的时间，设置成该文件在 Github Action 构建机器下的mtime。

   那么Hexo在渲染的时候就会将 post 的上次更新时间设置为该 post 上次 git commit 的时间了（我们将每一次 post 文件的 git commit 都视作发/更新 post）。

   这部分工作实际已经包含于上文的 deploy 脚本中，这里单独拎出来：

   ```bash
   # fetch mtime from git log
   git ls-files -z source/_posts |
       while read -d '' path; do
       if [[ $path == *.md ]]; then
           mtime=$(git log -1 --format="@%ct" $path)
           touch -d $mtime $path
           echo "change $path mtime to $mtime"
       fi
       done
   ```

   **注意：**

   checkout action 默认是 shallow clone，但我们需要拉取全量 git log 才能读到正确的 last commit time，所以进行如下配置：

   ```yaml
   - name: Checkout to main
           with:
             submodules: recursive
             # 拉取全量 log
             fetch-depth: 0
           uses: actions/checkout@v2
   ```

   checkout action 的更多信息：

   https://github.com/actions/checkout