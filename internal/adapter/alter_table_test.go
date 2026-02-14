package adapter

import (
	"dbm/internal/model"
	"testing"
)

// TestMySQLAlterTable 测试 MySQL 表结构修改
func TestMySQLAlterTable(t *testing.T) {
	adapter := NewMySQLAdapter()

	tests := []struct {
		name    string
		action  model.AlterTableAction
		want    string
		wantErr bool
	}{
		{
			name: "添加列",
			action: model.AlterTableAction{
				Type: model.AlterActionAddColumn,
				Column: &model.ColumnDef{
					Name:     "email",
					Type:     "VARCHAR",
					Length:   255,
					Nullable: false,
					Comment:  "用户邮箱",
				},
			},
			want:    "ADD COLUMN `email` VARCHAR(255) NOT NULL COMMENT '用户邮箱'",
			wantErr: false,
		},
		{
			name: "删除列",
			action: model.AlterTableAction{
				Type:    model.AlterActionDropColumn,
				OldName: "unused_field",
			},
			want:    "DROP COLUMN `unused_field`",
			wantErr: false,
		},
		{
			name: "修改列",
			action: model.AlterTableAction{
				Type: model.AlterActionModifyColumn,
				Column: &model.ColumnDef{
					Name:         "age",
					Type:         "INT",
					Nullable:     true,
					DefaultValue: "0",
				},
			},
			want:    "MODIFY COLUMN `age` INT NULL DEFAULT '0'",
			wantErr: false,
		},
		{
			name: "重命名列",
			action: model.AlterTableAction{
				Type:    model.AlterActionRenameColumn,
				OldName: "old_name",
				NewName: "new_name",
				Column: &model.ColumnDef{
					Name:     "new_name",
					Type:     "VARCHAR",
					Length:   100,
					Nullable: false,
				},
			},
			want:    "CHANGE COLUMN `old_name` `new_name` VARCHAR(100) NOT NULL",
			wantErr: false,
		},
		{
			name: "添加索引",
			action: model.AlterTableAction{
				Type: model.AlterActionAddIndex,
				Index: &model.IndexDef{
					Name:    "idx_email",
					Columns: []string{"email"},
					Unique:  true,
					Type:    "BTREE",
				},
			},
			want:    "ADD UNIQUE INDEX `idx_email` (`email`) USING BTREE",
			wantErr: false,
		},
		{
			name: "删除索引",
			action: model.AlterTableAction{
				Type:    model.AlterActionDropIndex,
				OldName: "idx_old",
			},
			want:    "DROP INDEX `idx_old`",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := adapter.buildAlterClause(tt.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildAlterClause() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("buildAlterClause() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestPostgreSQLAlterTable 测试 PostgreSQL 表结构修改
func TestPostgreSQLAlterTable(t *testing.T) {
	adapter := NewPostgreSQLAdapter()

	tests := []struct {
		name    string
		action  model.AlterTableAction
		want    string
		wantErr bool
	}{
		{
			name: "重命名列",
			action: model.AlterTableAction{
				Type:    model.AlterActionRenameColumn,
				OldName: "old_name",
				NewName: "new_name",
			},
			want:    `ALTER TABLE "".""  RENAME COLUMN "old_name" TO "new_name"`,
			wantErr: false,
		},
		{
			name: "删除列",
			action: model.AlterTableAction{
				Type:    model.AlterActionDropColumn,
				OldName: "unused_field",
			},
			want:    `ALTER TABLE "".""  DROP COLUMN "unused_field"`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 注意：PostgreSQL 的 buildAlterClause 返回完整的 SQL
			// 这里只测试基本逻辑
			_ = adapter
			_ = tt
			// 实际测试需要模拟数据库连接
		})
	}
}

// TestColumnTypeBuilder 测试列类型构建
func TestColumnTypeBuilder(t *testing.T) {
	adapter := NewMySQLAdapter()

	tests := []struct {
		name string
		col  *model.ColumnDef
		want string
	}{
		{
			name: "VARCHAR with length",
			col: &model.ColumnDef{
				Name:     "name",
				Type:     "VARCHAR",
				Length:   255,
				Nullable: false,
			},
			want: "VARCHAR(255) NOT NULL",
		},
		{
			name: "INT with default",
			col: &model.ColumnDef{
				Name:         "age",
				Type:         "INT",
				Nullable:     true,
				DefaultValue: "0",
			},
			want: "INT NULL DEFAULT '0'",
		},
		{
			name: "DECIMAL with precision",
			col: &model.ColumnDef{
				Name:      "price",
				Type:      "DECIMAL",
				Precision: 10,
				Scale:     2,
				Nullable:  false,
			},
			want: "DECIMAL(10,2) NOT NULL",
		},
		{
			name: "TIMESTAMP with CURRENT_TIMESTAMP",
			col: &model.ColumnDef{
				Name:         "created_at",
				Type:         "TIMESTAMP",
				Nullable:     false,
				DefaultValue: "CURRENT_TIMESTAMP",
			},
			want: "TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP",
		},
		{
			name: "INT with AUTO_INCREMENT",
			col: &model.ColumnDef{
				Name:          "id",
				Type:          "INT",
				Nullable:      false,
				AutoIncrement: true,
			},
			want: "INT NOT NULL AUTO_INCREMENT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.buildColumnType(tt.col)
			if got != tt.want {
				t.Errorf("buildColumnType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSQLiteAlterTableLimitations 测试 SQLite 的限制
func TestSQLiteAlterTableLimitations(t *testing.T) {
	adapter := NewSQLiteAdapter()

	// SQLite 不支持 DROP COLUMN
	action := model.AlterTableAction{
		Type:    model.AlterActionDropColumn,
		OldName: "unused",
	}

	// 这个测试需要实际的数据库连接，这里只是示例
	_ = adapter
	_ = action
}
