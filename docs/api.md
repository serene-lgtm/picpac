# picpac API Summary

本文件面向前端开发者和前端 agent，用于快速了解API接口的功能、调用方式、endpoint及参数等信息。

如果接口实现发生变化，必须同步更新本文件。

## Project Summary

picpac 是一个个人物品管理手机 app 的后端服务。

当前技术栈：
- 后端：Golang + Gin
- 前端：Flutter
- 数据库：MongoDB
- 图片存储：腾讯云 COS
- API 风格：RESTful

## Formal APIs

### Create Item

`POST /api/v1/item`

用途：
- 创建一个用户私有的 item
- 如果上传图片，后端会先上传到腾讯云 COS，再把图片 URL 存入 MongoDB
- 新创建的 item 会默认写入 `created` 状态

请求类型：
- `multipart/form-data`

请求字段：
- `user_id`: string，可选。用户系统接入后会恢复为必填或从登录态获取
- `name`: string，必填
- `description`: string，可选
- `image`: 文件，可选

成功响应：

```json
{
  "id": "6821c0c1f1b2f4d5a6b7c8d9",
  "user_id": "6821c0c1f1b2f4d5a6b7c8d1",
  "name": "黑色双肩包",
  "description": "日常出差用",
  "source_image_url": "https://xxx.cos.../items/item_6821c0c1f1b2f4d5a6b7c8d9/source.jpg",
  "image_thumbnail_url": "",
  "ai_rendered_image_url": "",
  "status": "created"
}
```

失败响应：
- `400`: 缺少 `name`，`user_id` 非法，或上传文件不是有效图片
- `502`: 图片上传失败
- `500`: 创建 item 失败

### List Items

`GET /api/v1/item`

用途：
- 查询当前用户的全部 item
- 当前阶段 `user_id` 可选；不传时返回全部未删除 item
- 默认按创建时间倒序返回
- 已逻辑删除的 item 不会出现在列表中

请求参数：
- `user_id`: string，可选，放在 query string 中。用户系统接入后会恢复为必填或从登录态获取

成功响应：

```json
{
  "items": [
    {
      "id": "6821c0c1f1b2f4d5a6b7c8d9",
      "user_id": "6821c0c1f1b2f4d5a6b7c8d1",
      "name": "黑色双肩包",
      "description": "日常出差用",
      "source_image_url": "https://xxx.cos.../items/item_6821c0c1f1b2f4d5a6b7c8d9/source.jpg",
      "image_thumbnail_url": "",
      "ai_rendered_image_url": "",
      "status": "created"
    }
  ]
}
```

空列表响应：

```json
{
  "items": []
}
```

失败响应：
- `400`: `user_id` 不是合法 ObjectID
- `500`: 查询 item 列表失败

### Get Item

`GET /api/v1/item/:item_id`

用途：
- 根据 `item_id` 读取单个 item 详情
- 如果 item 已被逻辑删除，则按不存在处理

路径参数：
- `item_id`: string，必填，item 主键

成功响应：

```json
{
  "id": "6821c0c1f1b2f4d5a6b7c8d9",
  "user_id": "6821c0c1f1b2f4d5a6b7c8d1",
  "name": "黑色双肩包",
  "description": "日常出差用",
  "source_image_url": "https://xxx.cos.../items/item_6821c0c1f1b2f4d5a6b7c8d9/source.jpg",
  "image_thumbnail_url": "",
  "ai_rendered_image_url": "",
  "status": "created"
}
```

失败响应：
- `400`: 缺少 `item_id`，或 `item_id` 不是合法 ObjectID
- `404`: item 不存在
- `500`: 查询 item 失败

### Update Item

`PUT /api/v1/item/:item_id`

用途：
- 更新单个 item 的名称、描述和可选图片
- 如果上传新图片，会覆盖 `source_image_url`
- 如果 item 已被逻辑删除，则不允许更新

请求类型：
- `multipart/form-data`

路径参数：
- `item_id`: string，必填，item 主键

请求字段：
- `name`: string，必填
- `description`: string，可选
- `image`: 文件，可选

成功响应：

```json
{
  "id": "6821c0c1f1b2f4d5a6b7c8d9",
  "user_id": "6821c0c1f1b2f4d5a6b7c8d1",
  "name": "黑色双肩包升级版",
  "description": "更新后的描述",
  "source_image_url": "https://xxx.cos.../items/item_6821c0c1f1b2f4d5a6b7c8d9/source.png",
  "image_thumbnail_url": "",
  "ai_rendered_image_url": "",
  "status": "created"
}
```

失败响应：
- `400`: 缺少 `name`，`item_id` 非法，或上传文件不是有效图片
- `404`: item 不存在
- `502`: 图片上传失败
- `500`: 更新 item 失败

### Delete Item

`DELETE /api/v1/item/:item_id`

用途：
- 逻辑删除单个 item
- 删除后会把 `status` 置为 `deleted`，不会真的从 MongoDB 中移除

路径参数：
- `item_id`: string，必填，item 主键

成功响应：

```json
{
  "deleted": true
}
```

失败响应：
- `400`: 缺少 `item_id`，或 `item_id` 不是合法 ObjectID
- `404`: item 不存在
- `500`: 删除 item 失败

### Create Pack

`POST /api/v1/pack`

用途：
- 创建一个用户的 pack，用于规划一次打包清单
- `user_id` 当前阶段可选；用户系统接入后会恢复为必填或从登录态获取
- `items` 当前只校验 ID 格式，不校验 item 是否属于当前用户；用户系统接入后会补充权限校验
- 新创建的 pack 会默认写入 `created` 状态

请求类型：
- `application/json`

请求字段：
- `name`: string，必填
- `user_id`: string，可选。用户系统接入后会恢复为必填或从登录态获取
- `description`: string，可选
- `items`: string array，可选，item id 列表

请求示例：

```json
{
  "name": "日本出差",
  "user_id": "6821c0c1f1b2f4d5a6b7c8d1",
  "description": "东京 5 天商务行程",
  "items": [
    "6821c0c1f1b2f4d5a6b7c8d9"
  ]
}
```

成功响应：

```json
{
  "id": "6821c0c1f1b2f4d5a6b7c8e0",
  "user_id": "6821c0c1f1b2f4d5a6b7c8d1",
  "name": "日本出差",
  "description": "东京 5 天商务行程",
  "items": [
    "6821c0c1f1b2f4d5a6b7c8d9"
  ],
  "status": "created"
}
```

失败响应：
- `400`: 缺少 `name`，`user_id` 非法，或 `items` 中存在非法 item id
- `500`: 创建 pack 失败

### List Packs

`GET /api/v1/pack`

用途：
- 查询 pack 列表
- 当前阶段 `user_id` 可选；不传时返回全部未删除 pack
- `status` 是内部状态，不支持作为 query 参数过滤
- 默认按创建时间倒序返回
- 已逻辑删除的 pack 不会出现在列表中

请求参数：
- `user_id`: string，可选，放在 query string 中。用户系统接入后会恢复为必填或从登录态获取

成功响应：

```json
{
  "packs": [
    {
      "id": "6821c0c1f1b2f4d5a6b7c8e0",
      "user_id": "6821c0c1f1b2f4d5a6b7c8d1",
      "name": "日本出差",
      "description": "东京 5 天商务行程",
      "items": [
        "6821c0c1f1b2f4d5a6b7c8d9"
      ],
      "status": "created"
    }
  ]
}
```

空列表响应：

```json
{
  "packs": []
}
```

失败响应：
- `400`: `user_id` 不是合法 ObjectID
- `500`: 查询 pack 列表失败

### Get Pack

`GET /api/v1/pack/:pack_id`

用途：
- 根据 `pack_id` 读取单个 pack 详情
- 如果 pack 已被逻辑删除，则按不存在处理
- `status` 是内部状态，不支持作为 query 参数过滤

路径参数：
- `pack_id`: string，必填，pack 主键

成功响应：

```json
{
  "id": "6821c0c1f1b2f4d5a6b7c8e0",
  "user_id": "6821c0c1f1b2f4d5a6b7c8d1",
  "name": "日本出差",
  "description": "东京 5 天商务行程",
  "items": [
    "6821c0c1f1b2f4d5a6b7c8d9"
  ],
  "status": "created"
}
```

失败响应：
- `400`: 缺少 `pack_id`，或 `pack_id` 不是合法 ObjectID
- `404`: pack 不存在
- `500`: 查询 pack 失败

### Update Pack

`PUT /api/v1/pack/:pack_id`

用途：
- 更新单个 pack 的完整可编辑字段
- 前端提交更新后的 `name`、`description`、`items`
- `name` 必填
- `description` 传空字符串表示清空描述
- `items` 传空数组表示清空 pack 内 item 列表
- 后端会保留 `id`、`user_id`、`status`、`created_at` 等系统字段，并更新 `updated_at`
- 如果 pack 已被逻辑删除，则不允许更新

请求类型：
- `application/json`

路径参数：
- `pack_id`: string，必填，pack 主键

请求字段：
- `name`: string，必填
- `description`: string，可选
- `items`: string array，可选，表示更新后的完整 item id 列表

请求示例：

```json
{
  "name": "日本出差升级版",
  "description": "东京 6 天商务行程",
  "items": [
    "6821c0c1f1b2f4d5a6b7c8d9"
  ]
}
```

成功响应：

```json
{
  "id": "6821c0c1f1b2f4d5a6b7c8e0",
  "user_id": "6821c0c1f1b2f4d5a6b7c8d1",
  "name": "日本出差升级版",
  "description": "东京 6 天商务行程",
  "items": [
    "6821c0c1f1b2f4d5a6b7c8d9"
  ],
  "status": "created"
}
```

失败响应：
- `400`: 缺少 `name`，`pack_id` 非法，或 `items` 中存在非法 item id
- `404`: pack 不存在
- `500`: 更新 pack 失败

### Delete Pack

`DELETE /api/v1/pack/:pack_id`

用途：
- 逻辑删除单个 pack
- 删除后会把 `status` 置为 `deleted`，不会真的从 MongoDB 中移除

路径参数：
- `pack_id`: string，必填，pack 主键

成功响应：

```json
{
  "deleted": true
}
```

失败响应：
- `400`: 缺少 `pack_id`，或 `pack_id` 不是合法 ObjectID
- `404`: pack 不存在或已被逻辑删除
- `500`: 删除 pack 失败

## Planned Domain APIs

后续仍计划补充以下正式接口：
- User authentication
