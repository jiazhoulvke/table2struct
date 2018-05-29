# table2struct #

我写golang代码时最烦的就是为MySQL数据库定义相应的struct，这种机械化的任务当然是要交给机器来完成啦。table2struct可以一次性生成整个数据库里所有表的struct，也可以选择性的生成某几个表的struct，然后根据你的需求稍加修改就能用了。

## 安装 ##

```bash
go get github.com/jiazhoulvke/table2struct
```

## 使用说明 ##

### 基本应用 ###

先来看看table2struct支持的参数:

```
Usage of table2struct:
      --db_host string        数据库ip地址 (default "127.0.0.1")
      --db_name string        数据库名
      --db_port int           数据库端口 (default 3306)
      --db_pwd string         数据库密码 (default "root")
      --db_user string        数据库用户名 (default "root")
      --int64                 是否将tinyint、smallint等类型也转换int64
      --mapping strings       强制将字段名转换成指定的名称。如--mapping foo:Bar,则表中叫foo的字段在golang中会强制命名为Bar
      --mapping_file string   字段名映射文件
      --output string         输出路径,默认为当前目录
      --package_name string   包名 (default "models")
      --query string          查询数据库字段名转换后的golang字段名并立即退出
      --skip_if_no_prefix     当表名不包含指定前缀时跳过不处理
      --table_prefix string   表名前缀
      --tag_gorm              是否生成gorm的tag
      --tag_gorm_type         是否将type包含进gorm的tag (default true)
      --tag_json              是否生成json的tag (default true)
      --tag_sqlx              是否生成sqlx的tag
      --tag_xorm              是否生成xorm的tag
      --tag_xorm_type         是否将type包含进xorm的tag (default true)
```

比如你有一个名叫mydatabase的数据库，里面有一个user表：

```sql
CREATE TABLE `user` (
  `id` int(8) NOT NULL AUTO_INCREMENT,
  `username` varchar(255) NOT NULL,
  `password` varchar(255) NOT NULL,
  `email` varchar(255) DEFAULT NULL,
  `age` int(10) unsigned DEFAULT NULL,
  `address` varchar(255) DEFAULT NULL,
  `status` tinyint(4) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB;
```

通过运行`table2struct --db_name mydatabase`就可以生成一个user.go文件:

```go
package models

//User user
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Age      uint   `json:"age"`
	Address  string `json:"address"`
	Status   int8   `json:"status"`
}

//TableName user
func (t *User) TableName() string {
	return "user"
}
```

介绍几个需要注意的参数：

- `--int64` 
 强制把所有整型字段全部声明为int64,比如上面示例中的Status为`Status int8`,加入参数--int64=true后，生成的字段就会是`Status int64`
- `--tag_json` 
 默认启用，会在struct的tag里增加`json:"字段名"`
- 同理，`--tag_sqlx`、`--tag_xorm`、`--tag_gorm`可以分别生成对应框架需要的tag


### 转换结果查询 ###

假如你还不想真正生成字段，只是想预览一下数据库里的字段会变成什么名字，就可以用`table2struct --query [表名.]字段名` 进行查询，比如：

```bash
$ table2struct --query table1.foo

table1.foo => table1.Foo
```


### 字段映射 ###

有时对于一些自动转换的字段名不满意，比如user表中有一个username字段，自动转换出来的将会是Username,但我想要它转成UserName，那该怎么办呢？这时就可以通过`--mapping`参数来强制将username转换成UserName。

```bash
$ table2struct --mapping username:UserName --query username

username => UserName
```

--mapping 这个参数是可以无限制的增加的，也就是说你可以这样:

```bash
table2struct --mapping foo1:foo2 --mapping bar1:bar2 --mapping baz1:baz2
```

左边的字段是可以带上表名的，比如这样:

```bash
$ table2struct --mapping table1.foo1:foo2 --query foo1

table1.foo1 = table1.Foo1 

$ table2struct --mapping table1.foo1:foo2 --query table1.foo1

table1.foo1 => table1.foo2
```

假如需要映射的字段名很多的话，放在一个文件里显然更合适一点，这时就可以用`--mapping_file`这个参数了:

```bash
$ cat mapping.txt

foo:bar

$ table2struct --mapping_file mapping.txt --query foo

foo => bar
```

### 处理前缀 ###

有时我们的表名都带有统一的前缀，比如:

> google_table1  
> google_table2  
> google_table3  

这时生成的文件名是google_table1.go，结构名是GoogleTable1。然而我们需要它生成的文件名是table1.go，结构名是Table1，这时就可以用到`--table_prefix`这个参数了

```bash
$ table2struct --table_prefix google_
```

