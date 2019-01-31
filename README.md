# httpdebug
调试http请求及响应

# 项目初衷
> 项目中经常碰到一些诡异的问题，需要深入到http层面定位问题， 当碰到http层面大多开发都有些无从入手。该工具会打印http的请求及响应方便问题定位

# 使用方式 
## 参数说明
```
  -b string
        直接相应 BODY， 可以直接跟字符串，亦可接文件（@filepath）
        例如: -b "{\"name\":\"value\"}"
              -b @file
         (default "default body")
  -c int
        直接响应 code (默认200) (default 200)
  -h string
        直接响应 HEADE
        例如： -h "Connection: keep-alive,Content-Type: application/json"
         (default "Content-Type: application/json;charset=utf-8")
  -l string
        监听的端口 (default "8899")
  -p string
        过滤URL 支持正则表达式 (default ".*")
  -s    不打印请求响应内容
  -u string
        被代理的服务器 eg: 172.30.0.100:8080

```


## Server模式 
> 该模式下默认会启动httpServer 相应内容会从命令行参数中读取并返回给请求者。 该模式适用于服务端调用第三方而第三方并未尚未提供接口  
> `./httpdebug -b @body` 访问 127.0.0.1:8899/**/* 返回body文件中的内容 

## 代理模式 
> 代理模式工具会作为一个方向代理服务器。 该模式用户调试 http请求及相应内容   

> ./httpdebug -u www.test.com 
