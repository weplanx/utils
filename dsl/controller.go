package dsl

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/errors"
	"github.com/weplanx/utils/passlib"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Controller struct {
	CollectionService *Service
}

type CreateDto struct {
	// 集合命名
	Collection string `path:"collection" vd:"regexp('^[a-z_]+$');msg:'集合名称必须是小写字母与下划线'"`
	// 文档数据
	Data M `json:"data,required" vd:"len($)>0;msg:'文档不能为空数据'"`
	// Body.data 格式转换
	Format M `json:"format"`
}

// Create 新增文档
// @router /:collection [POST]
func (x *Controller) Create(ctx context.Context, c *app.RequestContext) {
	var dto CreateDto
	if err := c.BindAndValidate(&dto); err != nil {
		c.Error(err)
		return
	}

	// 数据转换
	if err := x.Transform(dto.Data, dto.Format); err != nil {
		c.Error(errors.New(err, errors.ErrorTypePublic, nil))
		return
	}
	dto.Data["create_time"] = time.Now()
	dto.Data["update_time"] = time.Now()

	r, err := x.CollectionService.Create(ctx, dto.Collection, dto.Data)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, r)
}

type BulkCreateDto struct {
	// 集合命名
	Collection string `path:"collection,required" vd:"regexp('^[a-z_]+$');msg:'集合名称必须是小写字母与下划线'"`
	// 批量文档数据
	Data []M `json:"data,required" vd:"len($)>0 && range($,len(#v)>0);msg:'批量文档不能存在空数据'"`
	// Body.data[*] 格式转换
	Format M `json:"format"`
}

// BulkCreate 批量新增文档
// @router /:collection/bulk-create [POST]
func (x *Controller) BulkCreate(ctx context.Context, c *app.RequestContext) {
	var dto BulkCreateDto
	if err := c.BindAndValidate(&dto); err != nil {
		c.Error(err)
		return
	}

	// 数据转换
	docs := make([]interface{}, len(dto.Data))
	for i, doc := range dto.Data {
		if err := x.Transform(doc, dto.Format); err != nil {
			c.Error(errors.New(err, errors.ErrorTypePublic, nil))
			return
		}
		doc["create_time"] = time.Now()
		doc["update_time"] = time.Now()
		docs[i] = doc
	}

	r, err := x.CollectionService.BulkCreate(ctx, dto.Collection, docs)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, r)
}

type SizeDto struct {
	// 集合命名
	Collection string `path:"collection,required" vd:"regexp('^[a-z_]+$');msg:'集合名称必须是小写字母与下划线'"`
	// 筛选条件
	Filter M `query:"filter"`
	// Query.filter 格式转换
	Format M `query:"format"`
}

// Size 获取文档总数
// @router /:collection/_size [GET]
func (x *Controller) Size(ctx context.Context, c *app.RequestContext) {
	var dto SizeDto
	if err := c.BindAndValidate(&dto); err != nil {
		c.Error(err)
		return
	}

	// 数据转换
	if err := x.Transform(dto.Filter, dto.Format); err != nil {
		c.Error(errors.New(err, errors.ErrorTypePublic, nil))
		return
	}

	size, err := x.CollectionService.Size(ctx, dto.Collection, dto.Filter)
	if err != nil {
		c.Error(err)
		return
	}

	c.Header("x-total", strconv.Itoa(int(size)))
	c.Status(http.StatusNoContent)
}

type FindDto struct {
	// 集合命名
	Collection string `path:"collection,required" vd:"regexp('^[a-z_]+$');msg:'集合名称必须是小写字母与下划线'"`
	// 分页大小（默认 100 自定义必须在1~1000之间 ）
	Pagesize int64 `header:"x-pagesize" vd:"$>=0 && $<=1000;msg:'分页数量必须在 1~1000 之间'"`
	// 分页页码
	Page int64 `header:"x-page" vd:"$>=0;msg:'页码必须大于 0'"`
	// 筛选条件
	Filter M `query:"filter"`
	// Query.filter 格式转换
	Format M `query:"format"`
	// 排序规则
	Sort M `query:"sort" vd:"range($,in(#v,-1,1));msg:'排序规则不规范'"`
	// 投影规则
	Keys M `query:"keys" vd:"range($,in(#v,0,1));msg:'投影规则不规范'"`
}

// Find 获取匹配文档
// @router /:collection [GET]
func (x *Controller) Find(ctx context.Context, c *app.RequestContext) {
	var dto FindDto
	if err := c.BindAndValidate(&dto); err != nil {
		c.Error(err)
		return
	}

	// 数据转换
	if err := x.Transform(dto.Filter, dto.Format); err != nil {
		c.Error(errors.New(err, errors.ErrorTypePublic, nil))
		return
	}

	size, err := x.CollectionService.Size(ctx, dto.Collection, dto.Filter)
	if err != nil {
		c.Error(err)
		return
	}

	// 默认分页数量 100
	if dto.Pagesize == 0 {
		dto.Pagesize = 100
	}

	// 默认页码 1
	if dto.Page == 0 {
		dto.Page = 1
	}

	var sort bson.D
	for key, value := range dto.Sort {
		sort = append(sort, bson.E{Key: key, Value: value})
	}

	// 默认倒序 ID
	if len(sort) == 0 {
		sort = bson.D{{Key: "_id", Value: -1}}
	}

	option := options.Find().
		SetLimit(dto.Pagesize).
		SetSkip((dto.Page - 1) * dto.Pagesize).
		SetProjection(dto.Keys).
		SetSort(sort).
		SetAllowDiskUse(true)

	data, err := x.CollectionService.Find(ctx, dto.Collection, dto.Filter, option)
	if err != nil {
		c.Error(err)
		return
	}

	c.Header("x-total", strconv.Itoa(int(size)))
	c.JSON(http.StatusOK, data)
}

type FindOneDto struct {
	// 集合命名
	Collection string `path:"collection,required" vd:"regexp('^[a-z_]+$');msg:'集合名称必须是小写字母与下划线'"`
	// 筛选条件
	Filter M `query:"filter,required" vd:"len($)>0;msg:'筛选条件不能为空'"`
	// Query.filter 格式转换
	Format M `query:"format"`
	// 投影规则
	Keys M `query:"keys" vd:"range($,in(#v,0,1));msg:'投影规则不规范'"`
}

// FindOne 获取单个文档
// @router /:collection/_one [GET]
func (x *Controller) FindOne(ctx context.Context, c *app.RequestContext) {
	var dto FindOneDto
	if err := c.BindAndValidate(&dto); err != nil {
		c.Error(err)
		return
	}

	// 数据转换
	if err := x.Transform(dto.Filter, dto.Format); err != nil {
		c.Error(errors.New(err, errors.ErrorTypePublic, nil))
		return
	}

	option := options.FindOne().
		SetProjection(dto.Keys)
	data, err := x.CollectionService.FindOne(ctx, dto.Collection, dto.Filter, option)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, data)
}

type FindByIdDto struct {
	// 集合命名
	Collection string `path:"collection,required" vd:"regexp('^[a-z_]+$');msg:'集合名称必须是小写字母与下划线'"`
	// 文档 ID
	Id string `path:"id,required" vd:"mongoId($);msg:'文档 ID 不规范'"`
	// 投影规则
	Keys M `query:"keys" vd:"range($,in(#v,0,1));msg:'投影规则不规范'"`
}

// FindById 获取指定 ID 的文档
// @router /:collection/:id [GET]
func (x *Controller) FindById(ctx context.Context, c *app.RequestContext) {
	var dto FindByIdDto
	if err := c.BindAndValidate(&dto); err != nil {
		c.Error(err)
		return
	}

	id, _ := primitive.ObjectIDFromHex(dto.Id)
	option := options.FindOne().
		SetProjection(dto.Keys)
	data, err := x.CollectionService.FindOne(ctx, dto.Collection, M{"_id": id}, option)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, data)
}

type UpdateDto struct {
	// 集合命名
	Collection string `path:"collection" vd:"regexp('^[a-z_]+$');msg:'集合名称必须是小写字母与下划线'"`
	// 筛选条件
	Filter M `query:"filter,required" vd:"len($)>0;msg:'筛选条件不能为空'"`
	// Query.filter 格式转换
	FFormat M `query:"format"`
	// 更新操作
	Data M `json:"data,required" vd:"len($)>0;msg:'更新操作不能为空'"`
	// Body.data 格式转换
	DFormat M `json:"format"`
}

// Update 局部更新匹配文档
// @router /:collection [PATCH]
func (x *Controller) Update(ctx context.Context, c *app.RequestContext) {
	var dto UpdateDto
	if err := c.BindAndValidate(&dto); err != nil {
		c.Error(err)
		return
	}

	// 数据转换
	if err := x.Transform(dto.Filter, dto.FFormat); err != nil {
		c.Error(errors.New(err, errors.ErrorTypePublic, nil))
		return
	}
	if err := x.Transform(dto.Data, dto.DFormat); err != nil {
		c.Error(errors.New(err, errors.ErrorTypePublic, nil))
		return
	}
	if _, ok := dto.Data["$set"]; !ok {
		dto.Data["$set"] = M{}
	}
	dto.Data["$set"].(M)["update_time"] = time.Now()

	r, err := x.CollectionService.Update(ctx, dto.Collection, dto.Filter, dto.Data)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, r)
}

type UpdateByIdDto struct {
	// 集合命名
	Collection string `path:"collection,required" vd:"regexp('^[a-z_]+$');msg:'集合名称必须是小写字母与下划线'"`
	// 文档 ID
	Id string `path:"id,required" vd:"mongoId($);msg:'文档 ID 不规范'"`
	// 更新操作
	Data M `json:"data,required" vd:"len($)>0;msg:'更新操作不能为空'"`
	// Body.data 格式转换
	Format M `json:"format"`
}

// UpdateById 局部更新指定 ID 的文档
// @router /:collection/:id [PATCH]
func (x *Controller) UpdateById(ctx context.Context, c *app.RequestContext) {
	var dto UpdateByIdDto
	if err := c.BindAndValidate(&dto); err != nil {
		c.Error(err)
		return
	}

	// 数据转换
	if err := x.Transform(dto.Data, dto.Format); err != nil {
		c.Error(errors.New(err, errors.ErrorTypePublic, nil))
		return
	}
	if _, ok := dto.Data["$set"]; !ok {
		dto.Data["$set"] = M{}
	}
	dto.Data["$set"].(M)["update_time"] = time.Now()

	id, _ := primitive.ObjectIDFromHex(dto.Id)
	r, err := x.CollectionService.UpdateById(ctx, dto.Collection, id, dto.Data)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, r)
}

type ReplaceDto struct {
	// 集合命名
	Collection string `path:"collection,required" vd:"regexp('^[a-z_]+$');msg:'集合名称必须是小写字母与下划线'"`
	// 文档 ID
	Id string `path:"id,required" vd:"mongoId($);msg:'文档 ID 不规范'"`
	// 文档数据
	Data M `json:"data,required" vd:"len($)>0;msg:'文档数据不能为空'"`
	// Body.data 格式转换
	Format M `json:"format"`
}

// Replace 替换指定 ID 的文档
// @router /:collection/:id [PUT]
func (x *Controller) Replace(ctx context.Context, c *app.RequestContext) {
	var dto ReplaceDto
	if err := c.BindAndValidate(&dto); err != nil {
		c.Error(err)
		return
	}

	// 数据转换
	if err := x.Transform(dto.Data, dto.Format); err != nil {
		c.Error(errors.New(err, errors.ErrorTypePublic, nil))
		return
	}
	dto.Data["create_time"] = time.Now()
	dto.Data["update_time"] = time.Now()

	id, _ := primitive.ObjectIDFromHex(dto.Id)
	r, err := x.CollectionService.Replace(ctx, dto.Collection, id, dto.Data)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, r)
}

type DeleteDto struct {
	// 集合命名
	Collection string `path:"collection,required" vd:"regexp('^[a-z_]+$');msg:'集合名称必须是小写字母与下划线'"`
	// 文档 ID
	Id string `path:"id,required" vd:"mongoId($);msg:'文档 ID 不规范'"`
}

// Delete 删除指定 ID 的文档
// @router /:collection/:id [DELETE]
func (x *Controller) Delete(ctx context.Context, c *app.RequestContext) {
	var dto DeleteDto
	if err := c.BindAndValidate(&dto); err != nil {
		c.Error(err)
		return
	}

	id, _ := primitive.ObjectIDFromHex(dto.Id)
	r, err := x.CollectionService.Delete(ctx, dto.Collection, id)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, r)
}

type BulkDeleteDto struct {
	// 集合命名
	Collection string `path:"collection,required" vd:"regexp('^[a-z_]+$');msg:'集合名称必须是小写字母与下划线'"`
	// 筛选条件
	Data M `json:"data,required" vd:"len($)>0;msg:'筛选条件不能为空'"`
	// Body.data 格式转换
	Format M `json:"format"`
}

// BulkDelete 批量删除匹配文档
// @router /:collection/bulk-delete [POST]
func (x *Controller) BulkDelete(ctx context.Context, c *app.RequestContext) {
	var dto BulkDeleteDto
	if err := c.BindAndValidate(&dto); err != nil {
		c.Error(err)
		return
	}

	// 数据转换
	if err := x.Transform(dto.Data, dto.Format); err != nil {
		c.Error(errors.New(err, errors.ErrorTypePublic, nil))
		return
	}

	r, err := x.CollectionService.BulkDelete(ctx, dto.Collection, dto.Data)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, r)
}

type SortDto struct {
	// 集合命名
	Collection string `path:"collection,required" vd:"regexp('^[a-z_]+$');msg:'集合名称必须是小写字母与下划线'"`
	// 文档 ID 数组
	Data []primitive.ObjectID `json:"data,required" vd:"len($)>0;msg:'数组必须均为文档 ID'"`
}

// Sort 排序文档
// @router /:collection/sort [POST]
func (x *Controller) Sort(ctx context.Context, c *app.RequestContext) {
	var dto SortDto
	if err := c.BindAndValidate(&dto); err != nil {
		c.Error(err)
		return
	}

	_, err := x.CollectionService.Sort(ctx, dto.Collection, dto.Data)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Transform 格式转换
func (x *Controller) Transform(data M, format M) (err error) {
	for path, spec := range format {
		keys, cursor := strings.Split(path, "."), data
		n := len(keys) - 1
		for _, key := range keys[:n] {
			if v, ok := cursor[key].(M); ok {
				cursor = v
			}
		}
		key := keys[n]
		if cursor[key] == nil {
			continue
		}
		switch spec {
		case "oid":
			// 转换为 ObjectId
			if cursor[key], err = primitive.ObjectIDFromHex(cursor[key].(string)); err != nil {
				return
			}
			break

		case "oids":
			// 转换为 ObjectId 数组
			oids := cursor[key].([]interface{})
			for i, id := range oids {
				if oids[i], err = primitive.ObjectIDFromHex(id.(string)); err != nil {
					return
				}
			}
			break

		case "date":
			if cursor[key], err = time.Parse(time.RFC3339, cursor[key].(string)); err != nil {
				return
			}
			break

		//case "date":
		//	// 转换为 ISODate
		//	if cursor[key], err = time.Parse(time.RFC1123, cursor[key].(string)); err != nil {
		//		return
		//	}
		//	break

		case "password":
			// 密码类型，转换为 Argon2id
			if cursor[key], err = passlib.Hash(cursor[key].(string)); err != nil {
				return
			}
			break
		}
	}
	return
}
