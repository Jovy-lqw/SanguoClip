# 华为渠道服使用指南

本指南适用于需要录制《三国杀移动版》华为渠道服对局的用户。华为渠道服必须使用华为账号登录，且游戏客户端应从华为应用市场安装；官网包、其他应用商店包和华为渠道服的角色数据通常不互通。

## 需要准备的软件

### 电脑端

| 软件 | 用途 | 安装方式 |
| --- | --- | --- |
| SanguoClip | 解析分享链接、控制录屏和剪辑 | 运行项目的 `setup` 脚本 |
| Android Platform Tools（ADB） | 连接安卓设备并自动录屏 | 由 `setup` 脚本安装 |
| FFmpeg | 探测和渲染视频 | 由 `setup` 脚本安装 |
| 安卓设备或模拟器 | 运行华为渠道客户端 | Windows x64 推荐雷电模拟器；macOS、Windows ARM64 推荐安卓真机 |

macOS 的 Android 官方 AVD 可以用于测试，但游戏兼容性、图形性能和账号登录稳定性通常不如真机。雷电模拟器仅支持 Windows Intel/AMD 处理器。

### 安卓设备端

按以下顺序安装：

1. **华为应用市场**：非华为设备从[华为应用市场官方页面](https://appgallery.huawei.com/)下载。不要使用第三方 APK 下载站。
2. **HMS Core（华为移动服务）**：打开华为应用市场，搜索 `HMS Core` 并安装或更新到最新版。
3. **《三国杀移动版》华为渠道客户端**：继续在华为应用市场内搜索并安装，不要用游戏官网包替代。

华为官方说明：

- [非华为设备如何使用华为账号登录游戏](https://consumer.huawei.com/cn/support/content/zh-cn00447295/)
- [非华为品牌设备如何安装 HMS Core](https://consumer.huawei.com/cn/support/content/zh-cn15905145/)

华为手机通常已经包含华为应用市场和 HMS Core，只需确认两者为最新版本，再从华为应用市场安装游戏。

## 完整流程

### 1. 初始化 SanguoClip

macOS：

```bash
./scripts/setup.sh
./scripts/run.sh
```

没有安卓真机并希望创建测试 AVD 时：

```bash
./scripts/setup.sh --with-emulator
./scripts/start-emulator.sh
```

Windows x64：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\setup.ps1 -WithEmulator
powershell -ExecutionPolicy Bypass -File .\scripts\start-emulator.ps1
powershell -ExecutionPolicy Bypass -File .\scripts\run.ps1
```

Windows ARM64 请执行不带 `-WithEmulator` 的初始化命令，并连接开启了 USB 调试的安卓真机。

### 2. 连接安卓设备

真机需要在开发者选项中开启 USB 调试，并在设备上确认电脑的调试授权。运行：

```bash
adb devices -l
```

设备状态必须为 `device`。`unauthorized` 表示仍需在安卓设备上确认授权，`offline` 表示需要重新连接或重启 ADB。

### 3. 安装华为运行环境

在安卓设备或模拟器的浏览器中打开[华为应用市场官方页面](https://appgallery.huawei.com/)，按页面提示下载并安装华为应用市场。

Android 可能提示允许浏览器“安装未知应用”。只对本次使用的浏览器临时授权，安装完成后可关闭该权限。

打开华为应用市场后：

1. 搜索 `HMS Core`，完成安装或更新。
2. 在“我的 > 头像”中登录与游戏角色对应的华为账号。
3. 搜索《三国杀移动版》，确认发布渠道为华为应用市场后安装。
4. 从华为应用市场首次启动游戏，审阅协议并完成游戏内授权。
5. 若出现启动公告，点击公告外的空白区域关闭；随后点击资源动画右侧的“开始游戏”，不要等待动画自动结束。

账号、密码、短信验证码和扫码确认必须直接在安卓设备中输入。不要将凭据写入 `.env`、项目文件、终端命令或聊天消息。

### 4. 检查安装结果

macOS：

```bash
./scripts/doctor.sh
```

Windows：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\doctor.ps1
```

诊断结果中的华为渠道组件应全部显示 `installed`，分享链接处理器应显示 `ready`：

```text
Huawei channel (emulator-5554)
  AppGallery  installed
  HMS Core    installed
  Sanguosha   installed
  deep link   ready
```

当前已知包名如下，通常不需要用户手动填写：

| 组件 | 包名 |
| --- | --- |
| 华为应用市场 | `com.huawei.appmarket` |
| HMS Core | `com.huawei.hwid` |
| 三国杀移动版华为渠道服 | `com.bf.sgs.hdexp.huawei` |

SanguoClip 会根据 `yokasgs://` 分享链接处理器自动发现客户端。若设备同时安装了多个渠道，开始录制时选择“华为渠道服”。

### 5. 录制分享对局

1. 打开 [SanguoClip 工作台](http://127.0.0.1:8787)。
2. 在“设备采集”中刷新设备。
3. 粘贴 `www.sanguosha.cn` 的对局分享链接。
4. 选择已登录设备和“华为渠道服”，开始录制。
5. 在游戏内播放完整回放。
6. 回到工作台停止录制，再创建剪辑并渲染。

ADB `screenrecord` 通常不包含游戏内部音频。需要原声时，请使用真机或模拟器自带的录制功能，然后把视频导入 SanguoClip。

## 常见问题

### 登录后没有原来的角色

先确认华为应用市场登录的是保存该角色数据的华为账号。华为账号、荣耀账号、官服账号和其他渠道账号不是同一账号体系。游戏登录时出现华为游戏中心欢迎提示和华为账号昵称，才表示当前客户端及登录链路正确。

### 华为应用市场无法登录

在华为应用市场内将 HMS Core 更新到最新版，并允许 HMS Core 在后台运行。仍失败时参考[华为官方排障说明](https://consumer.huawei.com/cn/support/content/zh-cn16004032/)。

### 工作台没有显示“华为渠道服”

依次检查：

1. `adb devices -l` 中设备状态是否为 `device`。
2. `doctor` 是否显示三个安卓组件均已安装。
3. 游戏是否确实从华为应用市场安装。
4. 游戏是否已经完成首次启动协议。
5. 关闭并重新打开游戏后，在工作台刷新设备。

如果 `Sanguosha` 为 `installed`、`deep link` 仍为 `missing`，可能是客户端版本尚未注册分享链接处理器。请先从华为应用市场更新游戏，再重新启动设备。

### 一直停在首次启动资源动画

资源动画不是必须完整播放的加载过程。若画面上覆盖启动公告，先点击公告外的空白区域将其关闭，再点击画面右侧的“开始游戏”。点击后等待客户端进入大厅或分享回放，不要持续等待动画自行结束。

### 模拟器内无法输入文字

先关闭模拟器并使用项目脚本重新启动。macOS AVD：

```bash
./scripts/start-emulator.sh
```

该脚本会启用硬件键盘转发。仍无法输入时，可改用模拟器软键盘，或在真机上完成账号登录。

### 存储空间不足

华为应用市场、HMS Core、游戏本体和游戏资源都需要设备存储空间。安装前先在安卓系统设置中检查可用空间；空间不足时优先清理不需要的客户端、下载文件和旧录像，不要直接修改正在使用的模拟器磁盘文件。
