package bit

import "gorm.io/gorm"

type Bit struct {
	tx *gorm.DB
}

func Initialize(db *gorm.DB) *Bit {
	return &Bit{tx: db}
}

type CrudOption func(*Crud)

func SetOrderBy(orders []string) CrudOption {
	return func(option *Crud) {
		option.orderBy = orders
	}
}

func (x *Bit) Crud(model interface{}, options ...CrudOption) *Crud {
	crud := &Crud{
		tx:    x.tx,
		model: model,
	}
	for _, apply := range options {
		apply(crud)
	}
	return crud
}
