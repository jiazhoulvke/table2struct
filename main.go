package main

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var (
	db                        *sqlx.DB
	useInt64                  bool
	commonInitialisms         = []string{"API", "ASCII", "CPU", "CSS", "DNS", "EOF", "GUID", "HTML", "HTTP", "HTTPS", "ID", "IP", "JSON", "LHS", "QPS", "RAM", "RHS", "RPC", "SLA", "SMTP", "SSH", "TLS", "TTL", "UI", "UID", "UUID", "URI", "URL", "UTF8", "VM", "XML", "XSRF", "XSS"}
	commonInitialismsReplacer *strings.Replacer

	dbHost      string
	dbPort      int
	dbUser      string
	dbPwd       string
	dbName      string
	output      string
	packageName string
	tagGORM     bool
	tagXORM     bool
	tagSQLX     bool
	tagJSON     bool
)

func init() {
	var commonInitialismsForReplacer []string
	for _, initialism := range commonInitialisms {
		commonInitialismsForReplacer = append(commonInitialismsForReplacer, strings.ToLower(initialism), initialism)
	}
	commonInitialismsReplacer = strings.NewReplacer(commonInitialismsForReplacer...)

	flag.BoolVar(&useInt64, "int64", false, "是否将tinyint、smallint等类型也转换int64")
	flag.StringVar(&dbHost, "db_host", "127.0.0.1", "数据库ip地址")
	flag.IntVar(&dbPort, "db_port", 3306, "数据库端口")
	flag.StringVar(&dbUser, "db_user", "root", "数据库用户名")
	flag.StringVar(&dbPwd, "db_pwd", "root", "数据库密码")
	flag.StringVar(&dbName, "db_name", "", "数据库名")
	flag.StringVar(&packageName, "package_name", "models", "包名")
	flag.StringVar(&output, "output", "", "输出路径,默认为当前目录")
	flag.BoolVar(&tagGORM, "tag_gorm", false, "是否生成gorm的tag")
	flag.BoolVar(&tagXORM, "tag_xorm", false, "是否生成xorm的tag")
	flag.BoolVar(&tagSQLX, "tag_sqlx", false, "是否生成sqlx的tag")
	flag.BoolVar(&tagJSON, "tag_json", true, "是否生成json的tag")
}

func main() {
	var err error
	flag.Parse()

	if output == "" {
		output, err = filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			fmt.Printf("获取当前路径失败")
			return
		}
	}

	gofmtCmd, err := exec.LookPath("gofmt")
	if err != nil {
		fmt.Printf("获取gofmt可执行文件的路径失败:%v\n", err)
		return
	}

	if _, statErr := os.Stat(output); statErr != nil {
		if os.IsNotExist(statErr) {
			fmt.Printf("错误的输入路径:%v", output)
			return
		}
	}
	if dbName == "" {
		fmt.Printf("请输入数据库名称")
		return
	}
	db, err = sqlx.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", dbUser, dbPwd, dbHost, dbPort, dbName))
	if err != nil {
		fmt.Printf("连接数据库失败:%v", err)
		return
	}
	defer db.Close()

	var tableNames []string
	if flag.NArg() > 0 {
		tableNames = flag.Args()
	} else {
		if tableNames, err = GetTableNames(); err != nil {
			fmt.Printf("读取表名失败:%v", err)
			return
		}
	}
	for _, tableName := range tableNames {
		table, err := GetTable(tableName)
		if err != nil {
			fmt.Printf("读取表%v失败:%v\n", tableName, err)
			return
		}
		tempFile, err := ioutil.TempFile("", "table2struct_")
		if err != nil {
			fmt.Printf("创建文件失败:%v\n", err)
			return
		}
		tempFile.WriteString(toStruct(table))
		cmd := exec.Command(gofmtCmd, tempFile.Name())
		out, err := cmd.Output()
		cmd.Stderr = os.Stderr
		if err != nil {
			fmt.Printf("执行格式化命令[%s %s]失败:%v\n", gofmtCmd, tempFile.Name(), err)
			return
		}
		tempFile.Close()
		os.Remove(tempFile.Name())
		if err = ioutil.WriteFile(filepath.Join(output, tableName+".go"), out, 0666); err != nil {
			fmt.Printf("保存文件失败:%v\n", err)
		}
	}

}

//Field 字段
type Field struct {
	Name          string
	Type          string
	Length        int
	DecimalDigits int
	IsUnsigned    bool
	EnableNull    bool
	IsPrimaryKey  bool
	Default       sql.NullString
}

//Table 表
type Table struct {
	Name    string
	Fields  []Field
	HasTime bool
}

//TableField 表字段属性
type TableField struct {
	Field      string         `db:"Field"`
	Type       string         `db:"Type"`
	Collation  sql.NullString `db:"Collation"`
	Null       sql.NullString `db:"Null"`
	Key        sql.NullString `db:"Key"`
	Default    sql.NullString `db:"Default"`
	Extra      sql.NullString `db:"Extra"`
	Privileges sql.NullString `db:"Privileges"`
	Comment    sql.NullString `db:"Comment"`
}

//toGoName 参考 github.com/jinzhu/gorm 的 ToDBName
func toGoName(dbName string) string {
	var value string
	for i, v := range dbName {
		if (v >= 'A' && v <= 'Z') || (v >= 'a' && v <= 'z') {
			value = dbName[i:]
			break
		}
	}
	value = commonInitialismsReplacer.Replace(value)
	buf := bytes.NewBufferString("")
	for i, v := range value[:len(value)-1] {
		if i > 0 {
			if v == '_' || v == '-' {
				continue
			}
			if value[i-1] == '_' {
				buf.WriteRune(unicode.ToUpper(v))
			} else {
				buf.WriteRune(v)
			}
		} else {
			buf.WriteRune(unicode.ToUpper(v))
		}
	}
	buf.WriteByte(value[len(value)-1])
	return buf.String()
}

//GetTableNames 获取所有表名
func GetTableNames() ([]string, error) {
	tables := make([]string, 0, 32)
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		return tables, err
	}
	var tableName string
	for rows.Next() {
		if err = rows.Scan(&tableName); err != nil {
			return tables, err
		}
		tables = append(tables, tableName)
	}
	return tables, nil
}

//GetTable 获取表
func GetTable(tableName string) (Table, error) {
	table := Table{
		Fields: make([]Field, 0, 16),
	}
	table.Name = tableName
	rows, err := db.Queryx("DESC `" + tableName + "`")
	if err != nil {
		return table, err
	}
	var tableField TableField
	for rows.Next() {
		if err = rows.StructScan(&tableField); err != nil {
			return table, err
		}
		field := ParseField(tableField)
		if field.Type == "time.Time" {
			table.HasTime = true
		}
		table.Fields = append(table.Fields, field)
	}
	return table, nil
}

const (
	tableTpl = `
package %s

%s

//%s %s
type %s struct {
%s
}

//TableName %s
func (t *%s) TableName() string {
	return "%s"
}`
)

//toStruct 将表转换为struct字符串
func toStruct(table Table) string {
	buf := bytes.NewBufferString("")
	for _, field := range table.Fields {
		buf.WriteString("\t" + toGoName(field.Name) + "\t" + field.Type)
		tags := make([]string, 0)
		if tagJSON {
			tags = append(tags, `json:"`+field.Name+`"`)
		}
		if tagSQLX {
			tags = append(tags, `db:"`+field.Name+`"`)
		}
		if tagGORM {
			tags = append(tags, `gorm:"column:`+field.Name+`"`)
		}
		if tagXORM {
			tags = append(tags, `xorm:"'`+field.Name+`'"`)
		}
		if len(tags) > 0 {
			tag := strings.Join(tags, " ")
			buf.WriteString(" `" + tag + "`")
		}
		buf.WriteRune('\n')
	}
	tableGoName := toGoName(table.Name)
	importString := "\n"
	if table.HasTime {
		importString = `
import (
	"time"
)`
	}
	return fmt.Sprintf(tableTpl, packageName, importString, tableGoName, table.Name, tableGoName, buf.String(), table.Name, tableGoName, table.Name)
}

//ParseField 解析字段
func ParseField(tField TableField) Field {
	var field Field
	attrs := strings.Split(tField.Type, " ")
	var t string
	for _, attr := range attrs {
		attr = strings.ToLower(attr)
		if attr == "unsigned" {
			field.IsUnsigned = true
		} else if strings.Contains(attr, "(") && strings.Contains(attr, ")") {
			l := strings.Index(attr, "(")
			if l > 0 {
				t = attr[0:l]
				//fmt.Println(attr[l+1 : len(attr)-1])
			}
		} else {
			t = attr
		}
	}
	if tField.Null.String == "NULL" {
		field.EnableNull = true
	}
	if tField.Key.String == "PRI" {
		field.IsPrimaryKey = true
	}
	field.Name = tField.Field
	field.Type = goType(t)
	field.Default = tField.Default
	if field.IsUnsigned {
		field.Type = "u" + field.Type
	}
	return field
}

func goType(dbType string) string {
	switch dbType {
	case "tinyint":
		if useInt64 {
			return "int64"
		}
		return "int8"
	case "smallint":
		fallthrough
	case "mediumint":
		fallthrough
	case "integer":
		fallthrough
	case "int":
		if useInt64 {
			return "int64"
		}
		return "int"
	case "bigint":
		return "int64"
	case "float":
		fallthrough
	case "double":
		fallthrough
	case "decimal":
		fallthrough
	case "numeric":
		return "float64"
	case "bool":
		return "bool"
	case "char":
		fallthrough
	case "varchar":
		fallthrough
	case "tinytext":
		fallthrough
	case "text":
		fallthrough
	case "mediumtext":
		fallthrough
	case "longtext":
		return "string"
	case "date":
		fallthrough
	case "datetime":
		fallthrough
	case "time":
		fallthrough
	case "timestamp":
		return "time.Time"
	default:
		return ""
	}
}

//ParseAttribute 解析属性
func ParseAttribute(attr string) {
	fmt.Println(attr)
}
