<div align="center">
  <img src="./assets/logo-go-intl.png" width="300" alt="go-intl 标志"/>
  <h1>go-intl</h1>
  <p><em>让 Go 终端、桌面和服务端应用共用一套翻译。</em></p>
  <p><a href="./README.md">English</a> | <strong>简体中文</strong></p>
</div>

先选好语言和命名空间，渲染时直接调用 `t("key")`。Bubble Tea 模型、MyGo 窗口和 HTTP 请求可以各自选择语言，共享同一个只读的翻译目录。

## 快速开始

核心库需要 Go 1.25 或更高版本。在本地仓库根目录执行以下命令，会在同级目录创建一个示例模块。Go 将从当前仓库读取 go-intl 源码。

```sh
checkout=$(pwd)
mkdir ../intl-demo
cd ../intl-demo
go mod init example.com/intl-demo
go mod edit -replace "github.com/waterkyuu/go-intl=$checkout"
go get github.com/waterkyuu/go-intl@v0.0.0
```

将下面的程序保存为示例模块的 `main.go`，然后运行 `go run .`。导入路径以 `go-intl` 结尾，代码中的包名为 `intl`。

```go
package main

import (
	"fmt"
	"log"

	"github.com/waterkyuu/go-intl"
)

func main() {
	catalog, err := intl.New("en", intl.Define("en", map[string]string{
		"Inbox.title": "Inbox",
		"Inbox.hello": "Hello, {name}!",
		"Inbox.count": "{n, plural, =0 {No unread messages} one {# unread message} other {# unread messages}}",
		"Inbox.state": "{online, select, true {Online} other {Offline}}",
	}))
	if err != nil {
		log.Fatal(err)
	}

	t := catalog.Translations("en", "Inbox")
	fmt.Println(t("title"))                              // Inbox
	fmt.Println(t("hello", intl.Values{"name": "Ada"}))  // Hello, Ada!
	fmt.Println(t("count", intl.Values{"n": 2}))         // 2 unread messages
	fmt.Println(t("state", intl.Values{"online": true})) // Online
}
```

## 翻译文件

在普通的 `.go` 文件中调用 `intl.Define` 即可定义翻译，它会复制传入的 map。也可以从磁盘加载翻译文件，或将它们嵌入可执行文件。文件名使用 BCP 47 语言标签，例如 `fr-CA.toml`。同一种语言的翻译可以拆成多个文件，每个键只定义一次。

下面的示例从 Go 源文件旁边的 `locales/en.json` 加载默认语言。其他语言文件也放在 `locales` 目录中。

```go
package translations

import (
	"embed"

	"github.com/waterkyuu/go-intl"
)

//go:embed locales/*
var files embed.FS

// Load compiles the embedded translations at application startup.
func Load() (*intl.Catalog, error) {
	sources, err := intl.LoadFS(files, "locales/*")
	if err != nil {
		return nil, err
	}
	return intl.New("en", sources...)
}
```

| 来源 | 定义同一个键的写法 |
| --- | --- |
| Go | `intl.Define("en", map[string]string{"Inbox.title": "Inbox"})` |
| JSON | `{"Inbox": {"title": "Inbox"}}` |
| TOML | `Inbox.title = "Inbox"` |
| YAML | `Inbox: {title: Inbox}` |
| CSV | 第一行为 `key,message`，接着写入一行 `Inbox.title,Inbox` |

JSON、TOML 和 YAML 都支持用点分隔的键。CSV 字段用双引号包裹后可以包含逗号和换行符。空字符串是有效的译文。解码失败时，错误信息会指出文件名。构建目录时若发现模板无效或键重复，错误信息还会给出语言和键。

## 语言选择、命名空间与错误处理

在客户端调用 `catalog.Locale("fr-CA")` 可以选择语言。服务端可以将 HTTP 请求的 `Accept-Language` 值传给 `catalog.Match(header)`。某个键缺少译文时，先查找已注册的 CLDR 父语言，再查找配置的默认语言及其已注册的父语言。消息中的复数和占位符遵循实际译文的语言规则，每个请求独立保存自己的语言选择。

```go
package web

import (
	"io"
	"net/http"

	"github.com/waterkyuu/go-intl"
)

// Inbox renders a title in the language negotiated for this request.
func Inbox(catalog *intl.Catalog) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		locale := catalog.Match(r.Header.Get("Accept-Language"))
		w.Header().Set("Content-Language", locale.Tag().String())
		w.Header().Add("Vary", "Accept-Language")
		t := locale.Translations("Inbox")
		_, _ = io.WriteString(w, t("title"))
	}
}
```

在终端或桌面应用中，可以把 `catalog.Translations(preference, "Inbox")` 保存在模型或窗口状态里。用户切换语言时重新绑定即可。如果应用支持热更新，可以先构建新的翻译目录，成功后再替换旧目录。

| 调用 | 结果 |
| --- | --- |
| `inbox := locale.Scope("Inbox")` | 绑定命名空间，随后可以调用 `Text`、`Format`、`Has` 和 `Raw` |
| `inbox.Scope("toolbar").Text("save")` | 与 `locale.Text("Inbox.toolbar.save")` 访问同一条译文 |
| `inbox.Format("count", intl.Values{"n": 3})` | 返回译文和错误。键缺失时可用 `errors.Is(err, intl.ErrMissingMessage)` 判断 |
| `inbox.Text("unknown")` | 查找或格式化失败时返回带命名空间的完整键 |
| `inbox.Raw("count")` | 返回原始 ICU 模板和表示键是否存在的布尔值 |
| `inbox.Has("count")` | 检查当前语言及回退语言中是否存在该键 |

## 消息语法

| 用途 | 模板或用法 |
| --- | --- |
| 命名插值 | `Hello, {name}` |
| 布尔或枚举分支 | `{role, select, admin {Manage} other {View}}` |
| 基数复数 | `{n, plural, =0 {Empty} one {# item} other {# items}}` |
| 序数 | `{n, selectordinal, one {#st} two {#nd} few {#rd} other {#th}}` |
| 复数偏移量 | `{n, plural, offset:1 =1 {Just you} one {You and one guest} other {You and # guests}}` |
| 数字 | `{price, number}`、`{n, number, integer}`、`{share, number, percent}` |
| 日期和时间 | 向 `{when, date, long}` 或 `{when, time, short}` 传入 Go 的 `time.Time` 值 |
| 富文本 | `Read <strong>{title}</strong>` 配合 `intl.Values{"title": "Inbox", "strong": func(s string) string { return "**" + s + "**" }}` |
| 转义 | `'{name}'` 输出带花括号的字面文本，`''` 输出一个单引号 |

`select`、`plural` 和 `selectordinal` 都需要提供 `other` 分支，分支内可以继续嵌套消息。精确匹配（如 `=1`）使用原始数量，复数类别和 `#` 使用扣除偏移量后的数量。只需提供实际选中分支所用的参数。富文本回调返回字符串，适合添加终端样式。输出 HTML 时，应按插入位置对文本做相应的转义。

选定语言后，也可以直接调用 `Number`、`Percent`、`Currency`、`Date`、`Time`、`LanguageName`、`List` 和 `RelativeTime`。日期保留传入 `time.Time` 的时区。`Currency` 接受 ISO 4217 货币代码，遇到未知代码会返回错误。`List` 使用完整形式的连词连接各项。`RelativeTime` 按经过的时间计算，一天取 24 小时，一个月取 365 天的十二分之一。

用 `locale.RightToLeft()` 判断当前语言和书写系统是否需要从右向左布局。

## 翻译目录工具

从仓库根目录执行以下命令，可以用客户端示例中的英西双语翻译试用提取、检查和合并。

```sh
catalogs=$(mktemp -d)
go run ./cmd/go-intl extract -locale en ./examples/clients/internal/counter > "$catalogs/en.json"
go run ./cmd/go-intl extract -locale es ./examples/clients/internal/counter > "$catalogs/es.json"
go run ./cmd/go-intl check -base "$catalogs/en.json" "$catalogs/es.json"
go run ./cmd/go-intl merge -base "$catalogs/en.json" "$catalogs/es.json"
```

提取命令解析 Go 语法，支持导入别名。语言标签必须是字面量，翻译表必须是 `map[string]string` 字面量，其中的键和值也要写成字符串字面量。直接在程序中定义翻译时可以使用 Go 表达式。`check` 验证模板并报告缺失的键。`merge` 保留目标语言已有的翻译，用源语言文本补齐缺失的键，并将扁平 JSON 输出到标准输出。

## 运行客户端和服务端示例

两个客户端示例位于独立的 Go 模块中，共用一份英文和西班牙文翻译，核心库可以独立于 UI 框架使用。该示例模块需要 Go 1.27.3 或更高版本。两套界面都有无需打开窗口的测试。

```sh
cd examples/clients
go run ./bubbletea -locale es
go run ./mygo -locale es
go test ./...
```

在仓库根目录运行 `go test -run ExampleCatalog_Match -v` 可以执行 HTTP 语言协商示例。运行 `go test -race ./...` 可以测试核心库和命令行工具，并启用数据竞争检测。

## 性能

下表是三轮测试的中位数，单位为纳秒/次。测试环境为 Apple M3 Pro、Go 1.27.1（darwin/arm64）。英文目录包含 1,024 个静态键、一条问候语和一条复数消息。对比双方均复用已绑定的翻译器和参数 map。

| 操作 | go-intl（ns/op） | go-i18n（ns/op） | 每次分配数（go-intl / go-i18n） |
| --- | ---: | ---: | ---: |
| 重复读取同一个静态键 | 12.9 | 116.8 | 0 / 0 |
| 轮流读取不同静态键 | 15.8 | 125.8 | 0 / 0 |
| 插入姓名，输出 `Hello, Ada!` | 66.2 | 288.2 | 3 / 7 |
| 按英文复数规则输出 `7 items` | 81.2 | 336.5 | 2 / 8 |

在视图或请求开始时绑定一次语言和命名空间，可以省去每次翻译时的语言协商。每次新建参数 map、格式化带千位分隔符的数字或改用其他语言，都会影响耗时。完整报告还包含这些场景的测量结果、启动成本、并发吞吐量和复现命令。[性能报告](./PERFORMANCE.md)

翻译函数的调用方式和消息语法参考了 next-intl。Go 翻译目录的提取与合并沿用了 go-i18n 用户熟悉的工作流程。[next-intl 翻译 API](https://next-intl.dev/docs/usage/translations) · [go-i18n](https://github.com/nicksnyder/go-i18n)
