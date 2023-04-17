package dboperator

import (
	"context"
	"errors"
	"fmt"
	"github.com/onlyzzg/gentol/gormx"
)

func NewGPOperator() IOperator {
	return &GPOperator{}
}

type GPOperator struct{}

func (G GPOperator) Open(config *gormx.Config) error {
	return gormx.InitWithConfig(config)
}

func (G GPOperator) Ping(dbName string) error {
	return gormx.Ping(dbName)
}

func (G GPOperator) Close(dbName string) error {
	return gormx.Close(dbName)
}

func (G GPOperator) GetTablesUnderDB(ctx context.Context, dbName string) (dbTableMap map[string]*LogicDBInfo, err error) {
	dbTableMap = make(map[string]*LogicDBInfo)
	if dbName == "" {
		err = errors.New("empty dnName")
		return
	}
	gormDBTables := make([]*GormDBTable, 0)
	db, err := gormx.GetDB(dbName)
	if err != nil {
		return
	}
	db.WithContext(ctx).
		Raw("SELECT tb.schemaname as table_schema, " +
			"tb.tablename as table_name, " +
			"d.description as comments " +
			"FROM pg_tables tb " +
			"JOIN pg_class c ON c.relname = tb.tablename " +
			"LEFT JOIN pg_description d ON d.objoid = c.oid AND d.objsubid = '0' " +
			"WHERE schemaname <> 'information_schema' " +
			"AND tablename NOT LIKE 'pg%' " +
			"AND tablename NOT LIKE 'gp%' " +
			"AND tablename NOT LIKE 'sql_%' ").
		Find(&gormDBTables)
	if len(gormDBTables) == 0 {
		return
	}
	for _, row := range gormDBTables {
		if logicDBInfo, ok := dbTableMap[row.TableSchema]; !ok {
			dbTableMap[row.TableSchema] = &LogicDBInfo{
				SchemaName: row.TableSchema,
				TableInfoList: []*TableInfo{{
					TableName: row.TableName,
					Comment:   row.Comments,
				}},
			}
		} else {
			logicDBInfo.TableInfoList = append(logicDBInfo.TableInfoList,
				&TableInfo{
					TableName: row.TableName,
					Comment:   row.Comments,
				})
		}
	}
	return
}

func (G GPOperator) GetColumns(ctx context.Context, dbName string) (dbTableColMap map[string]map[string]*TableColInfo, err error) {
	dbTableColMap = make(map[string]map[string]*TableColInfo, 0)
	if dbName == "" {
		err = errors.New("empty dnName")
		return
	}
	gormTableColumns := make([]*GormTableColumn, 0)
	db, err := gormx.GetDB(dbName)
	if err != nil {
		return
	}
	db.WithContext(ctx).
		Raw("select " +
			"ic.table_schema table_schema, " +
			"ic.table_name table_name, " +
			"ic.column_name as column_name, " +
			"ic.data_type as data_type, " +
			"d.description as comments " +
			"from " +
			"information_schema.columns ic " +
			"JOIN pg_class c ON c.relname = ic.table_name " +
			"LEFT JOIN pg_description d " +
			"ON d.objoid = c.oid AND d.objsubid = ic.ordinal_position " +
			"where ic.table_name NOT LIKE 'pg%' " +
			"AND ic.table_name NOT LIKE 'gp%' " +
			"AND ic.table_name NOT LIKE 'sql_%' " +
			"AND ic.table_schema <> 'information_schema'").
		Find(&gormTableColumns)
	if len(gormTableColumns) == 0 {
		return
	}

	for _, row := range gormTableColumns {
		if dbTableColInfoMap, ok := dbTableColMap[row.TableSchema]; !ok {
			dbTableColMap[row.TableSchema] = map[string]*TableColInfo{
				row.TableName: {
					TableName: row.TableName,
					ColumnInfoList: []*ColumnInfo{{
						ColumnName: row.ColumnName,
						Comment:    row.Comments,
						DataType:   row.DataType,
					}},
				},
			}
		} else if tableColInfo, ok_ := dbTableColInfoMap[row.TableName]; !ok_ {
			dbTableColInfoMap[row.TableName] = &TableColInfo{
				TableName: row.TableName,
				ColumnInfoList: []*ColumnInfo{{
					ColumnName: row.ColumnName,
					Comment:    row.Comments,
					DataType:   row.DataType,
				}},
			}
		} else {
			tableColInfo.ColumnInfoList = append(tableColInfo.ColumnInfoList, &ColumnInfo{
				ColumnName: row.ColumnName,
				Comment:    row.Comments,
				DataType:   row.DataType,
			})
		}
	}
	return
}

func (G GPOperator) GetColumnsUnderTables(ctx context.Context, dbName, logicDBName string, tableNames []string) (tableColMap map[string]*TableColInfo, err error) {
	tableColMap = make(map[string]*TableColInfo, 0)
	if dbName == "" {
		err = errors.New("empty dnName")
		return
	}
	if len(tableNames) == 0 {
		err = errors.New("empty tableNames")
		return
	}

	gormTableColumns := make([]*GormTableColumn, 0)
	db, err := gormx.GetDB(dbName)
	if err != nil {
		return
	}
	db.WithContext(ctx).
		Raw("select "+
			"ic.table_schema table_schema, "+
			"ic.table_name table_name, "+
			"ic.column_name as column_name, "+
			"ic.data_type as data_type, "+
			"d.description as comments "+
			"from "+
			"information_schema.columns ic "+
			"JOIN pg_class c ON c.relname = ic.table_name "+
			"LEFT JOIN pg_description d "+
			"ON d.objoid = c.oid AND d.objsubid = ic.ordinal_position "+
			"where "+
			"ic.table_schema = ? "+
			"and ic.table_name in ?", logicDBName, tableNames).
		Find(&gormTableColumns)
	if len(gormTableColumns) == 0 {
		return
	}

	for _, row := range gormTableColumns {
		if tableColInfo, ok := tableColMap[row.TableName]; !ok {
			tableColMap[row.TableName] = &TableColInfo{
				TableName: row.TableName,
				ColumnInfoList: []*ColumnInfo{{
					ColumnName: row.ColumnName,
					Comment:    row.Comments,
					DataType:   row.DataType,
				}},
			}
		} else {
			tableColInfo.ColumnInfoList = append(tableColInfo.ColumnInfoList, &ColumnInfo{
				ColumnName: row.ColumnName,
				Comment:    row.Comments,
				DataType:   row.DataType,
			})
		}
	}
	return
}

func (G GPOperator) CreateSchema(ctx context.Context, dbName, schemaName, commentInfo string) (err error) {
	if dbName == "" {
		err = errors.New("empty dnName")
		return
	}
	if commentInfo == "" {
		commentInfo = schemaName
	}
	db, err := gormx.GetDB(dbName)
	if err != nil {
		return
	}
	err = db.WithContext(ctx).Exec("create schema if not exists " + schemaName).Error
	if err != nil {
		return
	}
	commentStr := fmt.Sprintf("comment on schema %s is '%s'", schemaName, commentInfo)
	err = db.WithContext(ctx).Exec(commentStr).Error
	if err != nil {
		return
	}
	return
}

func (G GPOperator) ExecuteDDL(ctx context.Context, dbName, ddlStatement string) (err error) {
	if dbName == "" {
		err = errors.New("empty dnName")
		return
	}
	db, err := gormx.GetDB(dbName)
	if err != nil {
		return
	}
	err = db.WithContext(ctx).Exec(ddlStatement).Error
	if err != nil {
		return
	}
	return
}
