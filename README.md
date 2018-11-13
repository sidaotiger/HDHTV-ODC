# HDHTV Operation Data Collector
收集operate表中的数据并上传到服务器

## 编译
**Linux:**
```
make build
```
**Windows:**
```
make windows
```

## 使用
1. 创建配置文件
```
cp config.example.json config.json
```
2. 修改 config.json 文件的内容
3. 创建 next-dump-time 文件
```
date +%Y%m%d%H > next-dump-time
```
4. 运行程序
