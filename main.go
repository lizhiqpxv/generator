package main

import (
	"bytes"
	"fmt"
	"go/format"
	"html/template"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"generator/dto"
)

// *********************************************** 配置代码开始 ***********************************************
// 文件名生成规则： 使用下面👇的 struct 名称做为前缀，加上对应的功能描述，如： instance_api.go
// 此结构体为生成代码的根据，必须包含 ID 字段， 不支持 bool 类型
// parameter => 表示是否需要做为参数
// required => 表示是否为必须的参数
// time => 表示是否为时间字段

const (
	// ProjectName 项目名称
	ProjectName = "manager"
)

// *********************************************** 配置代码结束 ***********************************************

// *********************************************** 以下代码请不要随便更改 ***********************************************
func main() {
	// instance 根据上面定义的结构体修改
	for _, v := range dto.StructMap {
		generate(ProjectName, v)
	}
	log.Println("Finish all")
}

func generate(projectName string, instance interface{}) {
	var (
		err       error
		generator = &Generate{ProjectName: projectName}
		t         = reflect.TypeOf(instance)
		v         = reflect.ValueOf(instance)
		fields    = make([]*Field, 0)
		src       []byte
		f         *os.File
	)
	if t.Kind() != reflect.Struct {
		log.Printf("is not a valid Instance struct, please use Instance struct instead \n")
	}

	generator.TitleName = t.Name()
	generator.FileName = Camel2Case(t.Name())
	generator.Char = "`"
	generator.Name = LeftToLower(t.Name())

	for i := 0; i < v.NumField(); i++ {
		var a = t.Field(i)
		typeName := a.Type.Name()
		if a.Type.Kind() == reflect.Slice {
			typeName = a.Type.String()
		}
		field := &Field{
			Name:      a.Name,
			Type:      typeName,
			Json:      a.Tag.Get("json"),
			Time:      a.Tag.Get("time"),
			Required:  a.Tag.Get("required"),
			Parameter: a.Tag.Get("parameter"),
			JsonTag:   a.Tag.Get("json"),
			Char:      "`",
		}
		fields = append(fields, field)
	}
	generator.Fields = fields

	for k, val := range m {
		var filename = fmt.Sprintf("../%s%s.%s", addr[k], generator.FileName, "go")
		if src, err = parse(val, generator); err != nil {
			log.Printf("generate %s error: %s", filename, err)
		}

		if !fileExists(filename) {
			if f, err = os.Create(filename); err != nil {
				log.Fatal(err)
			}
			if _, err = f.Write(src); err != nil {
				log.Fatal(err)
			}
			f.Close()
		}
	}
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

type Generate struct {
	ProjectName string
	TitleName   string
	Name        string
	FileName    string
	Char        string
	Fields      []*Field
}

type Field struct {
	Name      string
	Type      string
	Json      string
	Required  string
	Parameter string
	JsonTag   string
	Time      string
	Char      string
}

// addr 存储位置
var addr = map[string]string{
	"api":      "/server/web/v1/", // 接口存储位置
	"model":    "/model/",         // model 生成文件存储位置
	"entity":   "/model/entity/",
	"postgres": "/store/postgres/",
	"store":    "/store/",
	"bll":      "/bll/",
}

var m = map[string]string{
	"api":      apiTemplate,
	"model":    modelTemplate,
	"bll":      bllTemplate,
	"store":    interfaceTemplate,
	"postgres": storeTemplate,
	"entity":   entityTemplate,
}

var (
	importExistMap = map[string]struct{}{}
	lock           sync.RWMutex
)

func importNotExist(strType string) bool {
	lock.Lock()
	defer lock.Unlock()
	var ok bool
	if _, ok = importExistMap[strType]; ok {
		return false
	}

	importExistMap[strType] = struct{}{}
	return true
}

func formatParams(params ...string) (ret string) {
	for i := 0; i < len(params); i++ {
		ret = fmt.Sprintf("%v/%v", ret, params[i])
	}
	return
}

func parse(temp string, generator *Generate) ([]byte, error) {
	var (
		tmpl = template.New("")
		err  error
		p    *template.Template
		buf  = bytes.NewBuffer([]byte{})
		src  []byte
	)
	if p, err = tmpl.Funcs(template.FuncMap{
		"notExist": importNotExist,
		"format":   formatParams,
	}).Parse(temp); err != nil {
		return nil, err
	}

	if err = p.Execute(buf, generator); err != nil {
		return nil, err
	}
	newStr := strings.Replace(buf.String(), "|| {", "{", -1)
	if src, err = format.Source([]byte(newStr)); err != nil {
		return nil, err
	}
	return src, nil
}

func Camel2Case(name string) string {
	buffer := NewBuffer()
	for i, r := range name {
		if unicode.IsUpper(r) {
			if i != 0 {
				buffer.Append('_')
			}
			buffer.Append(unicode.ToLower(r))
		} else {
			buffer.Append(r)
		}
	}
	return buffer.String()
}

func LeftToLower(s string) string {
	if len(s) > 0 {
		return strings.ToLower(string(s[0])) + s[1:]
	}
	return s
}

func Ucfirst(str string) string {
	for i, v := range str {
		return string(unicode.ToUpper(v)) + str[i+1:]
	}
	return ""
}

func Lcfirst(str string) string {
	for i, v := range str {
		return string(unicode.ToLower(v)) + str[i+1:]
	}
	return ""
}

type Buffer struct {
	*bytes.Buffer
}

func NewBuffer() *Buffer {
	return &Buffer{Buffer: new(bytes.Buffer)}
}

func (b *Buffer) Append(i interface{}) *Buffer {
	switch val := i.(type) {
	case int:
		b.append(strconv.Itoa(val))
	case int64:
		b.append(strconv.FormatInt(val, 10))
	case uint:
		b.append(strconv.FormatUint(uint64(val), 10))
	case uint64:
		b.append(strconv.FormatUint(val, 10))
	case string:
		b.append(val)
	case []byte:
		b.Write(val)
	case rune:
		b.WriteRune(val)
	}
	return b
}

func (b *Buffer) append(s string) *Buffer {
	defer func() {
		if err := recover(); err != nil {
			log.Println("*****内存不够了！******")
		}
	}()
	b.WriteString(s)
	return b
}

var apiTemplate = `
package v1

import (
	"github.com/gin-gonic/gin"
	"{{.ProjectName}}/bll"
	"{{.ProjectName}}/model"
	"{{.ProjectName}}/server/web/middleware"
	"{{.ProjectName}}/utils"
)

var {{.TitleName}} = &{{.Name}}{}

func init() {
	// 注册路由
	RegisterRouter({{.TitleName}})
}


type {{.Name}} struct {}

// Init 初始化路由
func (a *{{.Name}}) Init (r *gin.RouterGroup) {
	g := r.Group("/{{.Name}}",  middleware.Auth())
	{
		g.POST("/create", a.create)
		g.POST("/update", a.update)
		g.POST("/list", a.list)
		g.POST("/delete", a.delete)
		g.POST("/detail", a.find)
	}
}

// create 创建
func (a *{{.Name}}) create(c *gin.Context) {
	var (
		in  = &model.{{.TitleName}}CreateRequest{}
		err error
	)

	if err = c.ShouldBindJSON(in); err != nil {
		c.Error(err)
		return
	}

	if err = bll.{{.TitleName}}.Create(c.Request.Context(), in); err != nil {
		c.Error(err)
		return
	}
	utils.ResponseOk(c, nil)
}

// update 更新
func (a *{{.Name}}) update(c *gin.Context) {
	var (
		in  = &model.{{.TitleName}}UpdateRequest{}
		err error
	)

	if err = c.ShouldBindJSON(in); err != nil {
		c.Error(err)
		return
	}

	if err = bll.{{.TitleName}}.Update(c.Request.Context(), in); err != nil {
		c.Error(err)
		return
	}
	utils.ResponseOk(c, nil)
}

// list 列表查询
func (a *{{.Name}}) list(c *gin.Context) {
	var (
		in  = &model.{{.TitleName}}ListRequest{}
		out  = &model.{{.TitleName}}ListResponse{}
		err error
	)

	if err = c.ShouldBindJSON(in); err != nil {
		c.Error(err)
		return
	}

	if out, err = bll.{{.TitleName}}.List(c.Request.Context(), in); err != nil {
		c.Error(err)
		return
	}
	utils.ResponseOk(c, out)
}

// list 列表查询
func (a *{{.Name}}) find(c *gin.Context) {
	var (
		in  = &model.{{.TitleName}}InfoRequest{}
		out  = &model.{{.TitleName}}Info{}
		err error
	)

	if err = c.ShouldBindJSON(in); err != nil {
		c.Error(err)
		return
	}

	if out, err = bll.{{.TitleName}}.Find(c.Request.Context(), in); err != nil {
		c.Error(err)
		return
	}
	utils.ResponseOk(c, out)
}

// delete 删除
func (a *{{.Name}}) delete(c *gin.Context) {
	var (
		in  = &model.{{.TitleName}}DeleteRequest{}
		err error
	)

	if err = c.ShouldBindJSON(in); err != nil {
		c.Error(err)
		return
	}

	if  err = bll.{{.TitleName}}.Delete(c.Request.Context(), in); err != nil {
		c.Error(err)
		return
	}
	utils.ResponseOk(c, nil)
}

`

var modelTemplate = `
{{$ID := "Id"}}
{{$create := "CreatedAt"}}
{{$true := "true"}}
{{$false := "false"}}

{{$string := "string"}}
{{$int64 := "int64"}}
{{$int32 := "int32"}}
{{$int := "int"}}
{{$point := "Point"}}
{{$strSlice := "pq.StringArray"}}
{{$int64Slice := "pq.Int64Array"}}
{{$projectName := .ProjectName}}
{{$titleName := .TitleName}}
{{$moduleName := "model"}}

package model

import (
	"{{.ProjectName}}/model/entity"

	{{range $value :=.Fields}}
		{{if and (eq $point .Type) (notExist (format $moduleName $point $titleName))}}
			"{{$projectName}}/model/po"	
		{{end}}

		{{if or (eq $strSlice .Type) (eq $int64Slice .Type) }}
			{{if notExist (format $moduleName "pq" $titleName)}}
				"github.com/lib/pq"
			{{end}}
		{{end}}
	{{end}}
)

// {{.TitleName}}CreateRequest 创建现场数据
type {{.TitleName}}CreateRequest struct {
{{range $value :=.Fields}}
	{{if ne $ID .Name}} 
		{{if eq .Parameter $true}}
			{{if eq .Required $true}}
				{{if eq $point .Type}} 
					{{.Name}} po.{{.Type}} {{.Char}}json:"{{$value.JsonTag}}" validate:"required"{{.Char}}
				{{else}} 
					{{.Name}} {{.Type}} {{.Char}}json:"{{$value.JsonTag}}" validate:"required"{{.Char}}
				{{end}}
			{{else}}
				{{if eq $point .Type}} 
					{{.Name}} po.{{.Type}} {{.Char}}json:"{{$value.JsonTag}}"{{.Char}}
				{{else}} 
					{{.Name}} {{.Type}} {{.Char}}json:"{{$value.JsonTag}}"{{.Char}}
				{{end}}
			{{end}}
		{{end}}
	{{end}}
{{end}}
}

// {{.TitleName}}UpdateRequest 更新现场数据
type {{.TitleName}}UpdateRequest struct {
	Id int64 {{.Char}}json:"id"{{.Char}}
{{range $value :=.Fields}}
	{{if eq $create .Name}} 
		{{.Name}} {{.Type}} {{.Char}}json:"{{$value.JsonTag}}"{{.Char}}
	{{else if eq .Parameter $true}}
		{{if eq .Required $true}}
			{{if eq $point .Type}} 
				{{.Name}} *po.{{.Type}} {{.Char}}json:"{{$value.JsonTag}}" validate:"required"{{.Char}}
			{{else}} 
				{{.Name}} *{{.Type}} {{.Char}}json:"{{$value.JsonTag}}" validate:"required"{{.Char}}
			{{end}}
		{{else}}
			{{if eq $point .Type}} 
				{{.Name}} *po.{{.Type}} {{.Char}}json:"{{$value.JsonTag}}"{{.Char}}
			{{else}} 
				{{.Name}} *{{.Type}} {{.Char}}json:"{{$value.JsonTag}}"{{.Char}}
			{{end}}
			
		{{end}}
	{{end}}
{{end}}
}

// {{.TitleName}}ListRequest 列表现场数据
type {{.TitleName}}ListRequest struct {
Index int {{.Char}}json:"index"{{.Char}}
Size int {{.Char}}json:"size"{{.Char}}
{{range $value :=.Fields}}
	{{if eq $ID .Name}} 
		{{.Name}} {{.Type}} {{.Char}}json:"{{$value.JsonTag}}"{{.Char}}
	{{else if eq .Parameter $true}}
		{{if eq .Required $true}}
			{{if eq $point .Type}} 
				{{.Name}} *po.{{.Type}} {{.Char}}json:"{{$value.JsonTag}}" validate:"required"{{.Char}}
			{{else}} 
				{{.Name}} *{{.Type}} {{.Char}}json:"{{$value.JsonTag}}" validate:"required"{{.Char}}
			{{end}}
		{{else}}
			{{if eq $point .Type}} 
				{{.Name}} *po.{{.Type}} {{.Char}}json:"{{$value.JsonTag}}"{{.Char}}
			{{else}} 
				{{.Name}} *{{.Type}} {{.Char}}json:"{{$value.JsonTag}}"{{.Char}}
			{{end}}
		{{end}}
	{{end}}
{{end}}
}


// {{.TitleName}}ListResponse 列表回包数据
type {{.TitleName}}ListResponse struct {
	Total int {{.Char}}json:"total"{{.Char}}
	List []*{{.TitleName}}Info {{.Char}}json:"list"{{.Char}}
}

// {{.TitleName}}InfoRequest 列表现场数据
type {{.TitleName}}InfoRequest struct {
{{range $value :=.Fields}}
	{{if eq $ID .Name}} 
		{{.Name}} {{.Type}} {{.Char}}json:"{{$value.JsonTag}}"{{.Char}}
	{{else if eq .Parameter $true}}
		{{if eq .Required $true}}
			{{if eq $point .Type}} 
				{{.Name}} *po.{{.Type}} {{.Char}}json:"{{$value.JsonTag}}" validate:"required"{{.Char}}
			{{else}} 
				{{.Name}} *{{.Type}} {{.Char}}json:"{{$value.JsonTag}}" validate:"required"{{.Char}}
			{{end}}
		{{else}}
			{{if eq $point .Type}} 
				{{.Name}} *po.{{.Type}} {{.Char}}json:"{{$value.JsonTag}}"{{.Char}}
			{{else}} 
				{{.Name}} *{{.Type}} {{.Char}}json:"{{$value.JsonTag}}"{{.Char}}
			{{end}}
		{{end}}
	{{end}}
{{end}}

}

// {{.TitleName}}Info 详细数据
type {{.TitleName}}Info struct {
{{range $value :=.Fields}}
	{{if eq $point .Type}} 
		{{.Name}} po.{{.Type}} {{.Char}}json:"{{$value.JsonTag}}"{{.Char}}
	{{else}}
		{{.Name}} {{.Type}} {{.Char}}json:"{{$value.JsonTag}}"{{.Char}}
	{{end}}
{{end}}
}

// {{.TitleName}}DeleteRequest 删除现场数据
type {{.TitleName}}DeleteRequest struct {
{{range $value :=.Fields}}
	{{if eq $ID .Name}} 
		{{.Name}} {{.Type}} {{.Char}}json:"{{$value.JsonTag}}"{{.Char}}
	{{end}}
{{end}}
}

// {{.TitleName}}sEntityToDto entity数据转换
func {{.TitleName}}sEntityToDto({{.Name}}s []*entity.{{.TitleName}}) []*{{.TitleName}}Info {
	out := make([]*{{.TitleName}}Info, 0, len({{.Name}}s))
	for _, c := range {{.Name}}s  {
		out = append(out, {{.TitleName}}EntityToDto(c))
	}
	return out
}

// {{.TitleName}}EntityToDto entity数据转换
func {{.TitleName}}EntityToDto(e *entity.{{.TitleName}}) *{{.TitleName}}Info {
	return &{{.TitleName}}Info{
		{{range $v :=.Fields}}
			{{.Name}}: {{if eq .Time $true}}e.{{.Name}}.Unix(),{{else}}e.{{.Name}},{{end}}
		{{end}}
	}
}
`

var entityTemplate = `
{{$ID := "Id"}}
{{$true := "true"}}
{{$string := "string"}}
{{$int64 := "int64"}}
{{$int32 := "int32"}}
{{$int := "int"}}
{{$text := "text"}}
{{$point := "Point"}}
{{$strSlice := "pq.StringArray"}}
{{$int64Slice := "pq.Int64Array"}}
{{$projectName := .ProjectName}}
{{$titleName := .TitleName}}
{{$moduleName := "entity"}}

package entity

import (
	{{range $value :=.Fields}}
		{{if and (eq $point .Type) (notExist (format $moduleName $point $titleName))}}
			"{{$projectName}}/model/po"	
		{{end}}

		{{if or (eq $strSlice .Type) (eq $int64Slice .Type) }}
			{{if notExist (format $moduleName "pq" $titleName)}}
				"github.com/lib/pq"
			{{end}}
		{{end}}
	{{end}}
)

type {{.TitleName}} struct {
{{range $value :=.Fields}}
	{{if eq $ID .Name}} 
		{{.Name}} {{.Type}} {{.Char}}gorm:"column:{{$value.JsonTag}};type:BIGINT;primary_key" json:"{{$value.JsonTag}}"{{.Char}}
	{{else if eq $true .Time}} 
		{{.Name}} time.Time {{.Char}}gorm:"column:{{$value.JsonTag}};type:TIMESTAMP" json:"{{$value.JsonTag}}"{{.Char}}
	{{else if eq $int64 .Type}} 
		{{.Name}} {{.Type}} {{.Char}}gorm:"column:{{$value.JsonTag}};type:BIGINT" json:"{{$value.JsonTag}}"{{.Char}}
	{{else if eq $string .Type}} 
		{{.Name}} {{.Type}} {{.Char}}gorm:"column:{{$value.JsonTag}};type:VARCHAR(255)" json:"{{$value.JsonTag}}"{{.Char}}
	{{else if eq $int32 .Type}} 
		{{.Name}} {{.Type}} {{.Char}}gorm:"column:{{$value.JsonTag}};type:TINYINT" json:"{{$value.JsonTag}}"{{.Char}}
	{{else if eq $int .Type}} 
		{{.Name}} {{.Type}} {{.Char}}gorm:"column:{{$value.JsonTag}};type:TINYINT" json:"{{$value.JsonTag}}"{{.Char}}
	{{else if eq $text .Type}} 
		{{.Name}} {{.Type}} {{.Char}}gorm:"column:{{$value.JsonTag}};type:TEXT" json:"{{$value.JsonTag}}"{{.Char}}
	{{else if eq $point .Type}} 
		{{.Name}} po.{{.Type}} {{.Char}}gorm:"column:{{$value.JsonTag}};type:POINT" json:"{{$value.JsonTag}}"{{.Char}}
	{{else if eq $strSlice .Type}} 
		{{.Name}} {{.Type}} {{.Char}}gorm:"column:{{$value.JsonTag}};type:VARCHAR[]" json:"{{$value.JsonTag}}"{{.Char}}
	{{else}} 
		{{.Name}} {{.Type}} {{.Char}}gorm:"column:{{$value.JsonTag}};type:JSON" json:"{{$value.JsonTag}}"{{.Char}}
	{{end}}
{{end}}
}

func (a *{{.TitleName}}) TableName() string {
	return "{{.FileName}}s"
}
`

var bllTemplate = `
{{$zero := 0}}
{{$empty := ""}}
{{$nil := "nil"}}
{{$true := "true"}}
{{$or := "||"}}
{{$ID := "Id"}}
{{$CreatedAt := "created_at"}}
{{$UpdatedAt := "updated_at"}}
{{$UserId := "user_id"}}

{{$string := "string"}}
{{$int64 := "int64"}}
{{$int32 := "int32"}}
{{$int := "int"}}

{{$projectName := .ProjectName}}

package bll 

import (
	"context"
	
	"{{.ProjectName}}/event"
	"{{.ProjectName}}/model"
	"{{.ProjectName}}/model/entity"
	"{{.ProjectName}}/store"
	"{{.ProjectName}}/store/postgres"
	"time"

	{{range $value :=.Fields}}
		{{if eq $value.JsonTag "user_id" }}
			"{{$projectName}}/auth"
		{{end}}
	{{end}}
)

type {{.Name}} struct{
	i{{.TitleName}} store.I{{.TitleName}}
}

var {{.TitleName}} = &{{.Name}}{
	i{{.TitleName}}: postgres.{{.TitleName}},
}

func (a *{{.Name}}) init()     func()   {
	return func() {}
}

func (a *{{.Name}}) onEvent(*event.Data) {}

// Create 创建
func (a *{{.Name}}) Create(ctx context.Context, in *model.{{.TitleName}}CreateRequest) error  {
	var (
		err error
	)
	
	{{range $v := .Fields}}
		{{if eq .Json $UserId}}
			// 获取用户Id
			in.UserId, _ = auth.ContextUserID(ctx)
		{{end}}
	{{end}}

	// 构建创建现场数据
	c := build{{.TitleName}}(in)
	_, err = a.i{{.TitleName}}.Create(ctx,c)
	return err
}

// Update 更新
func (a *{{.Name}}) Update(ctx context.Context, in *model.{{.TitleName}}UpdateRequest) error  {
	var (
		dict = make(map[string]interface{})
	)
	{{range $v := .Fields}}
		{{if eq .Parameter $true}}
			{{if ne .Required $true}}
			if in.{{.Name}} != nil {
				dict["{{.Json}}"] = in.{{.Name}}
			}
			{{end}}
		{{end}}
	{{end}}
	// do other update here
	updateAt := time.Now().Unix()
	in.UpdatedAt = &updateAt
	return a.i{{.TitleName}}.Update(ctx, in.Id, dict)
}

// Delete 删除
func (a *{{.Name}}) Delete(ctx context.Context, in *model.{{.TitleName}}DeleteRequest) error  {
	return a.i{{.TitleName}}.Delete(ctx,in.Id)
}

// List 列表查询
func (a *{{.Name}}) List(ctx context.Context, in *model.{{.TitleName}}ListRequest) (*model.{{.TitleName}}ListResponse, error)  {
	var (
		err error
		total int
		list []*entity.{{.TitleName}} 
		out = &model.{{.TitleName}}ListResponse{}
	)

	if total, list, err = a.i{{.TitleName}}.List(ctx,in); err != nil {
		return nil, err
	}
	
	out.Total = total
	out.List = model.{{.TitleName}}sEntityToDto(list)

	return out, nil
}

// Find 列表查询
func (a *{{.Name}}) Find(ctx context.Context, in *model.{{.TitleName}}InfoRequest) (*model.{{.TitleName}}Info, error)  {
	var (
		err error
		data *entity.{{.TitleName}} 
		out = &model.{{.TitleName}}Info{}
	)

	if data, err = a.i{{.TitleName}}.Find(ctx,in); err != nil {
		return nil, err
	}
	
	out = model.{{.TitleName}}EntityToDto(data)
	return out, nil
}

// build{{.TitleName}} 构建创建数据现场
func build{{.TitleName}}(in *model.{{.TitleName}}CreateRequest) *entity.{{.TitleName}} {
	// todo: check the entity is required
	return &entity.{{.TitleName}}{
		{{range $v :=.Fields}}
			{{if eq .Json $CreatedAt}}
				{{.Name}}:time.Now().Unix(),
			{{else if eq .Json $UpdatedAt}}
				{{.Name}}:time.Now().Unix(),
			{{else}}
				{{if ne .Name $ID}}{{.Name}}: {{if eq .Parameter $true}} {{if ne .Required $true}}in.{{.Name}},{{else}}in.{{.Name}},{{end}}{{else}}{{if eq .Type $string}}"",{{else}}0,{{end}}{{end}}{{end}}
			{{end}}
		{{end}}
	} 
}
`

var storeTemplate = `
{{$true := "true"}}
{{$string := "string"}}

package postgres

import (
	"context"
	"gorm.io/gorm"
	"{{.ProjectName}}/errors"
	"{{.ProjectName}}/model"
	"{{.ProjectName}}/model/entity"
)

var {{.TitleName}} = &{{.Name}}{}

type {{.Name}} struct{}

// Create 创建
func (a *{{.Name}}) Create(ctx context.Context, m *entity.{{.TitleName}}) (int64, error) {
	err := GetDB(ctx).Create(m).Error
	return m.Id, err
}

// Find 查找详情
func (a *{{.Name}}) Find(ctx context.Context, in *model.{{.TitleName}}InfoRequest ) (*entity.{{.TitleName}}, error ){
	e := &entity.{{.TitleName}}{}

	q := GetDB(ctx).Model(&entity.{{.TitleName}}{})

	if in.Id > 0 {
		err := q.First(&e, in.Id).Error
		return e, err
	}

	count := 0 
	{{range $v := .Fields}}
		{{if eq .Parameter $true}}
			{{if ne .Required $true}}
			if in.{{.Name}} != nil {
				{{if eq $string .Type}}
					q = q.Where("{{.Json}} like ?", in.{{.Name}}) 
				{{else}}
					q = q.Where("{{.Json}} = ?", in.{{.Name}}) 
				{{end}}
				count++
			}
			{{end}}
		{{end}}
	{{end}}

	if count == 0 {
		return e, errors.New("condition illegal")
	}

	err := q.First(&e).Error
	return e, err
}

// Update 更新
func (a *{{.Name}}) Update(ctx context.Context, id int64, dict map[string]interface{}) error {
	return GetDB(ctx).Model(&entity.{{.TitleName}}{}).Where("id = ?", id).Updates(dict).Error
}

// Delete 删除
func (a *{{.Name}}) Delete(ctx context.Context,id int64) error {
	return GetDB(ctx).Delete(&entity.{{.TitleName}}{}, id).Error
}

// List 列表查询
func (a *{{.Name}}) List(ctx context.Context,in *model.{{.TitleName}}ListRequest) (int, []*entity.{{.TitleName}}, error) {
	var (
		q        = GetDB(ctx).Model(&entity.{{.TitleName}}{})
		err      error
		total    int64
		{{.Name}}s []*entity.{{.TitleName}}
	)

	{{range $v := .Fields}}
		{{if eq .Parameter $true}}
			{{if ne .Required $true}}
			if in.{{.Name}} != nil {
				{{if eq $string .Type}}
					q = q.Where("{{.Json}} like ?", in.{{.Name}}) 
				{{else}}
					q = q.Where("{{.Json}} = ?", in.{{.Name}}) 
				{{end}}
				
			}
			{{end}}
		{{end}}
	{{end}}

	if err = q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if err = q.Limit(in.Size).Offset((in.Index - 1) * in.Size).Find(&{{.Name}}s).Error; err != nil {
		return 0, nil, err
	}
	return int(total), {{.Name}}s, nil
}

// ExecTransaction db事务执行
func (a *{{.Name}}) ExecTransaction(ctx context.Context, callback func(ctx context.Context) error) error {
	return GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		ctx = context.WithValue(ctx, DBCONTEXTKEY, tx)
		return callback(ctx)
	})
}
`

var interfaceTemplate = `
package store

import (
	"context"
	"{{.ProjectName}}/model"
	"{{.ProjectName}}/model/entity"
)

type I{{.TitleName}} interface {
	// Create 创建
	Create(ctx context.Context, e *entity.{{.TitleName}}) (int64, error)
	// Find 查找详情
	Find(ctx context.Context, in *model.{{.TitleName}}InfoRequest) (*entity.{{.TitleName}}, error)
	// Update 更新
	Update(ctx context.Context, id int64, updates map[string]interface{}) (error)
	// Delete 删除
	Delete(ctx context.Context, id int64) (error)
	// List 列表查询
	List(ctx context.Context, in *model.{{.TitleName}}ListRequest) (int, []*entity.{{.TitleName}}, error)
	// ExecTransaction db事务执行
	ExecTransaction(ctx context.Context, callback func(ctx context.Context) error) error 
}
`
