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
2. 修改config.json文件的内容
3. 创建nextdate文件
```
date +%Y%m%d > nextdate
```
4. 运行程序
