package dbr

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

type CaseUpdateStmt struct {
	runner
	EventReceiver
	Dialect
	Table        string
	PKey         string
	RunLen       int
	Column       []string
	Value        []CaseUpdateValue
	ReturnColumn []string

	custom		Custom //自定义参数
}

type CaseUpdateValue struct {
	Key string
	Val []interface{}
}

type CaseUpdateBuilder = CaseUpdateStmt

func (b *CaseUpdateStmt) Build(d Dialect, buf Buffer) error {
	//赋予批量更新默认最大上限
	if b.RunLen == 0 {
		b.RunLen = 1000
	}

	if b.Table == "" {
		return ErrTableNotSpecified
	}

	if len(b.Column) == 0 {
		return ErrColumnNotSpecified
	}
	WhereKey := []string{}
	buf.WriteString("UPDATE ")
	buf.WriteString(d.QuoteIdent(b.Table))
	buf.WriteString(" SET ")
	for i, col := range b.Column {
		if i > 0 {
			buf.WriteString(", ")
		}
		fieldname := strings.TrimSpace(col)
		expstr := ""
		//正则分割取值
		reg := regexp.MustCompile(`\s`)
		regstrs := reg.Split(col,2)
		if len(regstrs)==2{
			fieldname = strings.TrimSpace(regstrs[0])
			expstr = strings.TrimSpace(regstrs[1])
		}
		buf.WriteString(d.QuoteIdent(fieldname))
		buf.WriteString(" = ")
		if expstr!="" {
			buf.WriteString(d.QuoteIdent(fieldname))
			buf.WriteString(expstr)
		}
		buf.WriteString(" CASE ")
		buf.WriteString(d.QuoteIdent(b.PKey))
		for x, v := range b.Value {
			if x >= b.RunLen && b.RunLen > 0 {
				break
			}
			buf.WriteString(" WHEN ? THEN ? ")
			buf.WriteValue(v.Key)
			buf.WriteValue(v.Val[i])
		}
		buf.WriteString(" END ")
	}
	for x, v := range b.Value {
		if x >= b.RunLen && b.RunLen > 0 {
			break
		}
		WhereKey = append(WhereKey, v.Key)
	}
	//长度为10,Len为5
	if len(b.Value) > b.RunLen && b.RunLen != 0 {
		b.Value = b.Value[b.RunLen:]
	} else {
		b.Value = []CaseUpdateValue{}
	}

	buf.WriteString(" WHERE ")
	buf.WriteString(d.QuoteIdent(b.PKey))
	buf.WriteString(" IN (")
	i := 0
	for _, key := range WhereKey {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(" ? ")
		buf.WriteValue(key)
		i++
	}
	buf.WriteString(" )")
	return nil
}

func CaseUpdate(table string) *CaseUpdateStmt {
	return &CaseUpdateStmt{
		Table: table,
		Value: []CaseUpdateValue{},
	}
}

func (sess *Session) CaseUpdate(table string) *CaseUpdateStmt {
	b := CaseUpdate(table)
	b.runner = sess
	b.EventReceiver = sess.EventReceiver
	b.Dialect = sess.Dialect
	return b
}

func (tx *Tx) CaseUpdate(table string) *CaseUpdateStmt {
	b := CaseUpdate(table)
	b.runner = tx
	b.EventReceiver = tx.EventReceiver
	b.Dialect = tx.Dialect
	return b
}

// PKey		主键字段名
// column	更新字段名
func (b *CaseUpdateStmt) Columns(PKey string, column ...string) *CaseUpdateStmt {
	b.PKey = PKey
	b.Column = column
	return b
}

// PKValue	主键字段值
// value	对应更新字段值
func (b *CaseUpdateStmt) Values(PKValue interface{}, value ...interface{}) *CaseUpdateStmt {
	pk := fmt.Sprint(PKValue)
	for k, v := range b.Value {
		if v.Key == pk {
			b.Value[k].Val = value
		}
	}
	b.Value = append(b.Value, CaseUpdateValue{
		Key: fmt.Sprint(PKValue),
		Val: value,
	})
	return b
}

// 设置分批每次执行条数
func (b *CaseUpdateStmt) SetRunLen(i int) *CaseUpdateStmt {
	b.RunLen = i
	return b
}

// Returning specifies the returning columns for postgres.
func (b *CaseUpdateStmt) Returning(column ...string) *CaseUpdateStmt {
	b.ReturnColumn = column
	return b
}

//redis缓存数据
func (b *CaseUpdateStmt) Cache(cache customCache, key string) *CaseUpdateStmt {
    b.custom.isCache = true
    b.custom.cache = cache
    b.custom.cacheKey = key
    return b
}

//获取SQL
func (b *CaseUpdateStmt) GetSQL() (string, error) {
	b1 := *b
	b2 := &b1
	return getSQL(b2, b2.Dialect)
}

func (b *CaseUpdateStmt) Exec() error {
	var err error
	for len(b.Value) > 0 && err == nil {
		_, err = b.ExecContext(context.Background())
	}
	return err
}

func (b *CaseUpdateStmt) ExecContext(ctx context.Context) (sql.Result, error) {
    result, err := exec(ctx, b.runner, b.EventReceiver, b, b.Dialect, b.custom)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (b *CaseUpdateStmt) LoadContext(ctx context.Context, value interface{}) error {
	_, err := query(ctx, b.runner, b.EventReceiver, b, b.Dialect, value)
	return err
}

func (b *CaseUpdateStmt) Load(value interface{}) error {
	return b.LoadContext(context.Background(), value)
}
