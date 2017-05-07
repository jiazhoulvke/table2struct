# table2struct #

我写golang代码时最烦的就是为MySQL数据库定义相应的struct，这种机械化的任务当然是要交给机器来完成啦。table2struct可以一次性生成整个数据库里所有表的struct，也可以选择性的生成某几个表的struct，然后根据你的需求稍加修改就能用了。

## 安装

```bash
go get github.com/jiazhoulvke/table2struct
```

table2struct会调用gofmt来格式化代码，请确保gofmt可正常执行。

## 使用说明

先来看看table2struct支持的参数:

```
Usage of table2struct:
  -db_host string
    	数据库ip地址 (default "127.0.0.1")
  -db_name string
    	数据库名
  -db_port int
    	数据库端口 (default 3306)
  -db_pwd string
    	数据库密码 (default "root")
  -db_user string
    	数据库用户名 (default "root")
  -int64
    	是否将tinyint、smallint等类型也转换int64
  -output string
    	输出路径,默认为当前目录
  -package_name string
    	包名 (default "models")
  -tag_gorm
    	是否生成gorm的tag
  -tag_json
    	是否生成json的tag (default true)
  -tag_sqlx
    	是否生成sqlx的tag
  -tag_xorm
    	是否生成xorm的tag
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

通过运行`table2struct -db_name mydatabase`就可以生成一个user.go文件:

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

- -int64 
 强制把所有整型字段全部声明为int64,比如上面示例中的Status为`Status int8`,加入参数-int64=true后，生成的字段就会是`Status int64`
- -tag\_json 
 默认启用，会在struct的tag里增加`json:"字段名"`
- 同理，-tag\_sqlx、-tag\_xorm、-tag\_gorm可以分别生成对应框架需要的tag

最后展示一下比较全的用法:

```bash
> $ table2struct -db_host 127.0.0.1 -db_name mydatabase -db_port 3306 -db_user root -db_pwd root\
 -int64=true -output /tmp -package_name foo -tag_gorm=true -tag_xorm=true -tag_json=true -tag_sqlx=true user

> $ cat /tmp/user.go                                                        
package foo

//User user
type User struct {
	ID       int    `json:"id" db:"id" gorm:"column:id" xorm:"'id'"`
	Username string `json:"username" db:"username" gorm:"column:username" xorm:"'username'"`
	Password string `json:"password" db:"password" gorm:"column:password" xorm:"'password'"`
	Email    string `json:"email" db:"email" gorm:"column:email" xorm:"'email'"`
	Age      uint   `json:"age" db:"age" gorm:"column:age" xorm:"'age'"`
	Address  string `json:"address" db:"address" gorm:"column:address" xorm:"'address'"`
	Status   int8   `json:"status" db:"status" gorm:"column:status" xorm:"'status'"`
}

//TableName user
func (t *User) TableName() string {
	return "user"
}
```
