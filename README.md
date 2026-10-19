# SanguoClip

本地优先的三国杀对局采集与剪辑工具。它通过 ADB 控制已登录官方客户端的模拟器或安卓设备录屏，也支持直接导入本地录像；剪辑方案保存为时间段列表，再由 FFmpeg 确定性渲染。

## 当前能力

- 校验 `www.sanguosha.cn` 分享链接并提取 `yokasgs://` 唤起地址
- 枚举 ADB 设备、唤起官方客户端、开始和停止屏幕录制
- 导入 MP4、MOV、MKV、WebM、M4V 素材
- 视频预览、人工编辑保留时间段、异步 FFmpeg 渲染
- 渲染完成后直接在浏览器观看成片，也可下载 MP4
- 保存剪辑风格和示例素材
- 通过 OpenAI-compatible Chat Completions 根据创作备注生成 EDL
- 浏览器内环境诊断，缺少 ADB 时仍可走手动导入

自动发布到抖音和快手不在当前版本中。

## 平台支持

支持以下桌面系统：

- macOS 13 或更新版本：Apple Silicon、Intel
- Windows 10/11：x64、ARM64

运行源码需要 Go 1.26 或更高版本，视频探测和渲染需要 FFmpeg，自动采集额外需要 Android Platform Tools。仓库提供 macOS 和 Windows 初始化脚本，会安装缺失依赖、创建 `.env` 并构建程序；macOS 的 Android SDK 保存在项目 `.tools/` 中。

Windows x64 默认使用[雷电模拟器三国杀专区](https://www.ldmnq.com/app/4697/)；雷电官方只支持 Windows 的 Intel/AMD 处理器，不支持 macOS 或 Windows ARM64。macOS 继续使用 Android 官方 AVD，Windows ARM64 使用真实安卓设备。

## macOS 快速开始

前置条件只有 [Homebrew](https://brew.sh/)。拉取项目后执行：

```bash
./scripts/setup.sh
./scripts/run.sh
```

初始化脚本通过 Homebrew 安装 Go、FFmpeg 和 ADB，随后构建 `bin/sanguoclip`。打开 [http://127.0.0.1:8787](http://127.0.0.1:8787) 使用。

需要自动采集且本机没有安卓设备时，执行完整模拟器初始化：

```bash
./scripts/setup.sh --with-emulator
./scripts/start-emulator.sh
```

该命令会额外下载 JDK 21、Android Command-line Tools、Android 15 Google Play ARM64/x64 镜像，并创建 `SanguoClip_API35`。相关文件保存在 `.tools/`，约占 4 GB；首次执行会要求审阅 Android SDK 许可。自动化环境可显式传入 `--accept-licenses`。

雷电模拟器不提供 macOS 版本，因此不能在 Mac 上安装。需要更稳定的采集环境时，优先连接真实安卓设备。

Android 官方 AVD 默认使用兼容 Cocos 客户端的 `swangle` 渲染后端。需要切换时可设置
`SANGUOCLIP_EMULATOR_GPU`，例如 `SANGUOCLIP_EMULATOR_GPU=host ./scripts/start-emulator.sh`。

也可以通过 Makefile 执行相同流程：

```bash
make setup
make setup-emulator
make run
```

自定义监听地址或数据目录：

```bash
./scripts/run.sh --listen 127.0.0.1:9000 --data-dir ./my-data
```

## Windows 下载后快速开始

适用于没有 Go、FFmpeg 或 ADB 的新电脑：

1. 打开 [SanguoClip GitHub 仓库](https://github.com/Jovy-lqw/SanguoClip)，点击 **Code > Download ZIP**。
2. 解压 ZIP。不要直接在压缩包预览窗口中运行脚本。
3. 打开解压后的 `SanguoClip` 文件夹，在文件夹空白处按住 Shift 点击鼠标右键，选择“在终端中打开”。
4. 根据使用方式执行下面的一组命令。

只导入本地录像进行剪辑：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\setup.ps1
powershell -ExecutionPolicy Bypass -File .\scripts\run.ps1
```

Windows x64 需要自动录制游戏：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\setup.ps1 -WithEmulator
powershell -ExecutionPolicy Bypass -File .\scripts\start-emulator.ps1
powershell -ExecutionPolicy Bypass -File .\scripts\run.ps1
```

首次初始化可能需要数分钟。脚本会通过 `winget` 安装 Go、FFmpeg 和 ADB，创建本地 `.env`，构建 `bin\sanguoclip.exe`，最后运行环境诊断。`-WithEmulator` 还会从雷电三国杀专区下载专版安装器、校验 Windows 数字签名并打开官方安装向导；安装完成后自动配置雷电实例 0，并通过 `127.0.0.1:5555` 接入 ADB。雷电的安装许可和系统虚拟化授权必须由用户本人确认。

第一次启动雷电后，先从雷电游戏中心或对应渠道的官方应用商店安装并登录游戏。客户端准备完成后，以后可以直接启动游戏：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\start-emulator.ps1 -OpenGame
```

运行服务后打开 [http://127.0.0.1:8787](http://127.0.0.1:8787)。渲染完成的项目会同时显示“观看”和“下载”：观看会在浏览器弹窗内播放，下载会保存 MP4。

Windows ARM64 不支持雷电模拟器，初始化脚本会明确终止模拟器安装；此时请连接真实安卓设备，SanguoClip 主程序仍可原生运行。

Windows 的唯一前置条件是 `winget`。如果命令不存在，请先从 Microsoft Store 安装或更新“应用安装程序”，重新打开 PowerShell 后再执行初始化。若初始化提示依赖已经安装但 PATH 中不可用，同样关闭 PowerShell、重新打开后重试。

首次运行可用以下命令验收环境：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\doctor.ps1
```

`go`、`adb`、`ffmpeg` 和 `ffprobe` 应显示实际路径。使用雷电时还应显示 `ldconsole` 路径和 `127.0.0.1:5555 device`。若雷电无法启动，请先在 BIOS/UEFI 中启用 CPU 虚拟化，并确认没有被其他虚拟机软件独占。

自定义监听地址或数据目录：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\run.ps1 `
  -Listen 127.0.0.1:9000 `
  -DataDir .\my-data
```

macOS 和 Windows 都可以随时运行环境诊断：

```bash
./scripts/doctor.sh
```

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\doctor.ps1
```

## 设备采集

1. Windows x64 启动雷电模拟器；macOS 启动 Android AVD；也可以连接已开启 USB 调试的安卓设备。
2. 官服从[三国杀移动版官网](https://m.sanguosha.com/)下载 APK；渠道服必须从对应渠道的官方应用商店安装。
3. 通过脚本安装 APK，并在安卓设备中自行审阅协议、完成登录。

macOS：

```bash
./scripts/install-client.sh ~/Downloads/sgs.apk
```

Windows：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-client.ps1 `
  -ApkPath "$HOME\Downloads\sgs.apk"
```

Windows 检测到雷电后会优先通过 `ldconsole installapp` 安装并启动客户端；其他设备使用 ADB。不传 APK 路径时，脚本会打开官网。雷电三国杀专版通常会从内置游戏中心安装正版客户端。

安装脚本会从设备上已注册的 `yokasgs://` 处理器自动识别客户端包名，不再绑定官服包名。设备同时安装多个渠道时，可显式传入包名：

```bash
./scripts/install-client.sh ~/Downloads/sgs-channel.apk emulator-5554 com.example.channel
```

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-client.ps1 `
  -ApkPath "$HOME\Downloads\sgs-channel.apk" `
  -PackageName "com.example.channel"
```

开始采集前，工作台会列出设备上所有支持分享链接的客户端，必须选择本次要使用的游戏渠道。

### 华为渠道服

华为渠道服需要按以下顺序准备安卓设备：

1. 从华为官方页面安装华为应用市场。
2. 在华为应用市场内安装或更新 HMS Core。
3. 登录与游戏角色对应的华为账号。
4. 从华为应用市场安装《三国杀移动版》。
5. 首次启动若出现资源动画，先关闭公告，再点击右侧“开始游戏”，不要等待动画自动结束。
6. 运行 `doctor` 检查组件，再从工作台选择“华为渠道服”。

完整的软件清单、分平台命令、验收方法和常见问题见[华为渠道服使用指南](docs/huawei-channel.md)。

华为账号、密码、短信验证码或扫码确认只应直接输入到模拟器或安卓设备中，不要写入 `.env`、项目文件、终端命令或聊天消息。SanguoClip 不读取或保存登录凭据。

ADB `screenrecord` 在多数环境中不包含内部音频。需要游戏原声时，使用模拟器自带录制功能，并将结果导入工作台。

如不需要自动采集，可以跳过模拟器与 ADB，直接在工作台导入本地录像。

## 模型配置

模型配置只保存接口地址、模型名和 API Key 的环境变量名，不保存密钥。初始化脚本会从 `.env.example` 创建被 Git 忽略的 `.env`，可在其中设置：

```dotenv
SANGUOCLIP_MODEL_API_KEY=your-key
```

`scripts/run.sh` 和 `scripts/run.ps1` 会自动读取该文件，也仍然支持直接设置进程环境变量。

模型不会收到本地视频文件。当前版本只把素材时长、剪辑风格和人工填写的关键事件备注发送给模型，用于生成严格的 EDL 时间段建议。

## 数据与安全边界

- `data/state.json`：素材、任务、风格和模型非敏感配置
- `data/media/`：采集或导入的原始录像
- `data/exports/`：渲染成片
- 分享链接仅接受 HTTPS 的 `www.sanguosha.cn/share/index.html`
- 所有 ADB 和 FFmpeg 调用均使用参数数组，不经过 shell
- 雷电集成只使用官方 `ldconsole` 启动、配置和安装接口，不使用多开同步、宏或自动点击
- 工具不读取官方客户端登录令牌，不逆向协议，不自动上传视频

## 开发

```bash
make check
make doctor
make build-all
```

`make build-all` 在 `dist/` 生成 macOS Intel/Apple Silicon 和 Windows x64/ARM64 可执行文件。

当前生成物未进行 Apple Developer ID 或 Windows Authenticode 签名。向其他用户公开分发前，需要完成对应平台的代码签名，避免 Gatekeeper 或 SmartScreen 警告。

服务端使用 Go 标准库，前端资源编译进可执行文件，不需要 npm 安装或前端构建步骤。
