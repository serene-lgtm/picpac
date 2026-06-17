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

### Send Phone Code

`POST /api/v1/auth/phone/code`

用途：
- 发送手机号登录验证码
- 当前开发配置可使用固定验证码，生产环境必须接真实短信服务
- 同一手机号会受到重发间隔和每日发送次数限制

请求类型：
- `application/json`

请求字段：
- `phone`: string，必填。中国大陆11位手机号会标准化为 `+86` 格式；也支持传入带 `+` 的国际号码

请求示例：

```json
{
  "phone": "13800138000"
}
```

成功响应：

```json
{
  "sent": true
}
```

失败响应：
- `400`: 缺少 `phone`，或手机号格式非法
- `429`: 验证码发送过于频繁
- `500`: 创建验证码或发送验证码失败

### Phone Login

`POST /api/v1/auth/phone/login`

用途：
- 使用手机号和验证码登录
- 首次手机号登录会自动创建 `User` 和 `AuthIdentity(provider=phone)`
- 已存在手机号会复用原 User

请求类型：
- `application/json`

请求字段：
- `phone`: string，必填
- `code`: string，必填

请求示例：

```json
{
  "phone": "13800138000",
  "code": "123456"
}
```

成功响应：

```json
{
  "access_token": "...",
  "refresh_token": "...",
  "user": {
    "id": "6821c0c1f1b2f4d5a6b7c8d1",
    "display_name": "用户8000",
    "avatar_url": "",
    "status": "created"
  }
}
```

失败响应：
- `400`: 缺少 `phone`、缺少 `code`、手机号格式非法、验证码非法或超过尝试次数
- `404`: 已绑定身份对应的 User 不存在
- `409`: 创建登录身份发生冲突且无法恢复
- `500`: 创建 User、AuthIdentity 或 token 失败

### Refresh Auth Token

`POST /api/v1/auth/refresh`

用途：
- 使用 refresh token 换取新的 access token
- refresh token 保持不变，直到过期或 logout 后才失效
- refresh token 会存 hash，明文 token 只返回给客户端

请求类型：
- `application/json`

请求字段：
- `refresh_token`: string，必填

成功响应：

```json
{
  "access_token": "..."
}
```

失败响应：
- `400`: 缺少 `refresh_token`
- `401`: refresh token 非法、过期或已被 revoke
- `404`: User 不存在
- `500`: 查询 refresh token 或创建新 access token 失败

### Logout

`POST /api/v1/auth/logout`

用途：
- 废弃 refresh token
- access token 当前不落库，logout 后已签发的 access token 会自然过期

请求类型：
- `application/json`

请求字段：
- `refresh_token`: string，必填

成功响应：

```json
{
  "logged_out": true
}
```

失败响应：
- `400`: 缺少 `refresh_token`
- `401`: refresh token 非法
- `500`: revoke refresh token 失败

### Me

`GET /api/v1/me`

用途：
- 读取当前登录用户
- 需要在请求头传入 access token

请求头：
- `Authorization: Bearer <access_token>`

成功响应：

```json
{
  "id": "6821c0c1f1b2f4d5a6b7c8d1",
  "display_name": "用户8000",
  "avatar_url": "",
  "status": "created"
}
```

失败响应：
- `401`: 缺少 access token，access token 非法或已过期
- `404`: User 不存在
- `500`: 查询 User 失败

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
- 可通过 `q` 按 item name 或 description 做关键词子串匹配，主要用于中文 item 搜索
- 默认按创建时间倒序返回
- 已逻辑删除的 item 不会出现在列表中

请求参数：
- `user_id`: string，可选，放在 query string 中。用户系统接入后会恢复为必填或从登录态获取
- `q`: string，可选，按 item name 或 description 子串匹配；当前最大长度为 50 个字符，传空字符串会返回 `400`

搜索示例：

`GET /api/v1/item?q=充电`

说明：
- `q=充电` 可以匹配 `手机充电器`、`充电宝` 等名称，也可以匹配 description 中包含 `充电` 的 item
- 中文关键词不做分词，按原始子串匹配
- 英文关键词大小写不敏感
- `q` 搜索 `name` 和 `description`

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
- `400`: `user_id` 不是合法 ObjectID，或 `q` 为空/超过最大长度
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
- 可通过 `q` 按 pack name 或 description 做关键词子串匹配，主要用于中文 pack 搜索
- `status` 是内部状态，不支持作为 query 参数过滤
- 默认按创建时间倒序返回
- 已逻辑删除的 pack 不会出现在列表中

请求参数：
- `user_id`: string，可选，放在 query string 中。用户系统接入后会恢复为必填或从登录态获取
- `q`: string，可选，按 pack name 或 description 子串匹配；当前最大长度为 50 个字符，传空字符串会返回 `400`

搜索示例：

`GET /api/v1/pack?q=东京`

说明：
- `q=东京` 可以匹配 `日本出差` 这类 name，也可以匹配 description 中包含 `东京` 的 pack
- 中文关键词不做分词，按原始子串匹配
- 英文关键词大小写不敏感

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
- `400`: `user_id` 不是合法 ObjectID，或 `q` 为空/超过最大长度
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

### Create Checklist

`POST /api/v1/checklist`

用途：
- 创建一个 checklist
- `user_id` 当前阶段可选；用户系统接入后会恢复为必填或从登录态获取
- 新创建的 checklist 会默认写入 `created` 状态
- `items` 是 line item 列表；如果 `reference_type` 是 `item`，`reference_id` 必须是 item id，且不能传 `snapshot`；如果 `reference_type` 是 `snapshot`，`reference_id` 必须为空，且必须传 `snapshot.name`
- `reference_type` 是 `item` 时，后端会校验对应 item 存在且未被逻辑删除
- line item 初始状态统一为 `unchecked`

请求类型：
- `application/json`

请求字段：
- `user_id`: string，可选。用户系统接入后会恢复为必填或从登录态获取
- `name`: string，必填
- `description`: string，可选
- `target_date`: string，必填，格式为 `YYYY-MM-DD`
- `items`: object array，可选

请求示例：

```json
{
  "name": "日本出差 checklist",
  "description": "东京 5 天商务行程",
  "target_date": "2026-07-01",
  "items": [
    {
      "reference_type": "item",
      "reference_id": "6821c0c1f1b2f4d5a6b7c8d9"
    },
    {
      "reference_type": "snapshot",
      "reference_id": "",
      "snapshot": {
        "name": "临时雨伞"
      }
    }
  ]
}
```

成功响应：

```json
{
  "id": "6821c0c1f1b2f4d5a6b7c8e0",
  "user_id": "6821c0c1f1b2f4d5a6b7c8d1",
  "name": "日本出差 checklist",
  "description": "东京 5 天商务行程",
  "target_date": "2026-07-01",
  "items": [
    {
      "id": "6821c0c1f1b2f4d5a6b7c8e1",
      "reference_type": "item",
      "reference_id": "6821c0c1f1b2f4d5a6b7c8d9",
      "snapshot": null,
      "status": "unchecked"
    },
    {
      "id": "6821c0c1f1b2f4d5a6b7c8e2",
      "reference_type": "snapshot",
      "reference_id": "",
      "snapshot": {
        "name": "临时雨伞"
      },
      "status": "unchecked"
    }
  ],
  "status": "created"
}
```

失败响应：
- `400`: 缺少 `name`、`target_date`，`user_id` 非法，字段格式非法，line item 非法，或引用的 item 不存在/已删除
- `500`: 创建 checklist 失败

### List Checklists

`GET /api/v1/checklist`

用途：
- 查询 checklist 列表
- 当前阶段 `user_id` 可选；不传时返回全部未删除 checklist
- 可通过 `q` 按 checklist name 或 description 做关键词子串匹配
- 默认按创建时间倒序返回
- 已逻辑删除的 checklist 不会出现在列表中

请求参数：
- `user_id`: string，可选，放在 query string 中。用户系统接入后会恢复为必填或从登录态获取
- `q`: string，可选，按 checklist name 或 description 子串匹配；当前最大长度为 50 个字符，传空字符串会返回 `400`

成功响应：

```json
{
  "checklists": []
}
```

失败响应：
- `400`: `user_id` 不是合法 ObjectID，或 `q` 为空/超过最大长度
- `500`: 查询 checklist 列表失败

### Get Checklist

`GET /api/v1/checklist/:checklist_id`

用途：
- 根据 `checklist_id` 读取单个 checklist 详情
- 如果 checklist 已被逻辑删除，则按不存在处理

路径参数：
- `checklist_id`: string，必填，checklist 主键

成功响应：
- 同 Create Checklist 成功响应结构

失败响应：
- `400`: 缺少 `checklist_id`，或 `checklist_id` 不是合法 ObjectID
- `404`: checklist 不存在
- `500`: 查询 checklist 失败

### Update Checklist

`PUT /api/v1/checklist/:checklist_id`

用途：
- 更新单个 checklist 的 metadata
- 前端提交更新后的 `name`、`description`、`target_date`
- 后端会保留 `id`、`user_id`、`status`、`created_at` 等系统字段，并更新 `updated_at`
- `items` 不允许通过该接口更新；line item 需要使用 Add/Remove Checklist Line Items 接口修改
- 如果 checklist 已被逻辑删除，则不允许更新

请求类型：
- `application/json`

路径参数：
- `checklist_id`: string，必填，checklist 主键

请求字段：
- `name`: string，必填
- `description`: string，可选
- `target_date`: string，必填，格式为 `YYYY-MM-DD`

失败响应：
- `400`: 缺少 `name`、`target_date`，`checklist_id` 非法，字段格式非法，或请求体包含 `items`
- `404`: checklist 不存在
- `500`: 更新 checklist 失败

### Add Checklist Line Items

`POST /api/v1/checklist/:checklist_id/items`

用途：
- 向指定 checklist 批量增加 line item
- 新增 line item 会自动生成自己的 `id`
- 新增 line item 初始状态统一为 `unchecked`
- 如果 line item 的 `reference_type` 是 `item`，后端会校验对应 item 存在且未被逻辑删除
- 更新成功后会更新 checklist 的 `updated_at`

请求类型：
- `application/json`

路径参数：
- `checklist_id`: string，必填，checklist 主键

请求字段：
- `items`: object array，必填且不能为空
- `items[].reference_type`: string，必填，只支持 `item` 或 `snapshot`
- `items[].reference_id`: string。当 `reference_type` 为 `item` 时必填且必须是存在、未删除的 item id；当 `reference_type` 为 `snapshot` 时必须为空
- `items[].snapshot`: object。当 `reference_type` 为 `snapshot` 时必填
- `items[].snapshot.name`: string，当 `reference_type` 为 `snapshot` 时必填

请求示例：

```json
{
  "items": [
    {
      "reference_type": "item",
      "reference_id": "6821c0c1f1b2f4d5a6b7c8d9"
    },
    {
      "reference_type": "snapshot",
      "snapshot": {
        "name": "临时雨伞"
      }
    }
  ]
}
```

成功响应：
- 同 Create Checklist 成功响应结构，返回增加后的完整 checklist

失败响应：
- `400`: 缺少 `items`，`checklist_id` 非法，line item 非法，或引用的 item 不存在/已删除
- `404`: checklist 不存在
- `500`: 更新 checklist 失败

### Remove Checklist Line Items

`DELETE /api/v1/checklist/:checklist_id/items`

用途：
- 从指定 checklist 批量移除 line item
- 只有请求中的所有 `line_item_ids` 都属于当前 checklist 时才会更新
- 更新成功后会更新 checklist 的 `updated_at`

请求类型：
- `application/json`

路径参数：
- `checklist_id`: string，必填，checklist 主键

请求字段：
- `line_item_ids`: string array，必填且不能为空，值为 checklist line item 的 `id`

请求示例：

```json
{
  "line_item_ids": [
    "6821c0c1f1b2f4d5a6b7c8e1",
    "6821c0c1f1b2f4d5a6b7c8e2"
  ]
}
```

成功响应：
- 同 Create Checklist 成功响应结构，返回移除后的完整 checklist

失败响应：
- `400`: 缺少 `line_item_ids`，`checklist_id` 非法，或 `line_item_ids` 中存在非法 ObjectID
- `404`: checklist 不存在，或存在不属于该 checklist 的 line item id
- `500`: 更新 checklist 失败

### Update Checklist Line Item Status

`PATCH /api/v1/checklist/:checklist_id/items/:line_item_id/status`

用途：
- 更新指定 checklist 中单个 line item 的勾选状态
- 每次请求只更新一个 line item 的 `status`
- 更新成功后会更新 checklist 的 `updated_at`

请求类型：
- `application/json`

路径参数：
- `checklist_id`: string，必填，checklist 主键
- `line_item_id`: string，必填，checklist line item 主键

请求字段：
- `status`: string，必填，只允许 `checked` 或 `unchecked`

请求示例：

```json
{
  "status": "checked"
}
```

成功响应：
- 同 Create Checklist 成功响应结构，返回更新后的完整 checklist

失败响应：
- `400`: 缺少 `status`，`status` 非法，`checklist_id` 非法，或 `line_item_id` 非法
- `404`: checklist 不存在，或 line item 不属于该 checklist
- `500`: 更新 checklist line item status 失败，或读取更新后的 checklist 失败

### Delete Checklist

`DELETE /api/v1/checklist/:checklist_id`

用途：
- 逻辑删除单个 checklist
- 删除后会把 `status` 置为 `deleted`，不会真的从 MongoDB 中移除

路径参数：
- `checklist_id`: string，必填，checklist 主键

成功响应：

```json
{
  "deleted": true
}
```

失败响应：
- `400`: 缺少 `checklist_id`，或 `checklist_id` 不是合法 ObjectID
- `404`: checklist 不存在或已被逻辑删除
- `500`: 删除 checklist 失败

## Planned Domain APIs

后续仍计划补充以下正式接口：
- User authentication
