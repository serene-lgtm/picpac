# picpac API Summary

本文件面向前端开发者和前端 agent，用于快速了解API接口的功能、调用方式、endpoint及参数等信息。

如果接口实现发生变化，必须同步更新本文件。

## Project Summary

picpac 是一个个人物品管理手机 app 的后端服务。

当前技术栈：
- 后端：Golang + Gin
- 前端：Flutter
- 数据库：MongoDB
- 图片存储：阿里云 OSS
- API 风格：RESTful

图片 URL 约定：
- 接口响应中的 `avatar_url`、`source_image_url`、`image_thumbnail_url`、`ai_rendered_image_url` 是临时 signed URL，会过期。
- 前端不应长期持久化这些 URL；如果图片访问过期，应重新请求相关列表/详情/用户接口获取新 URL。
- 后端持久化 OSS object key，不把带 `Expires`、`OSSAccessKeyId`、`Signature` 的 URL 写入 MongoDB。

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
    "profile": {
      "username": "user8613800138000",
      "gender": "",
      "birthday": "",
      "avatar_url": "https://picpac.oss-cn-shanghai.aliyuncs.com/user-avatar/default.jpg?Expires=1783588103&OSSAccessKeyId=...&Signature=..."
    },
    "status": "created"
  }
}
```

失败响应：
- `400`: 缺少 `phone`、缺少 `code`、手机号格式非法、验证码非法或超过尝试次数
- `404`: 已绑定身份对应的 User 不存在
- `409`: 创建登录身份发生冲突且无法恢复
- `500`: 创建 User、AuthIdentity、token 或生成头像访问 URL 失败

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

### Delete Account

`DELETE /api/v1/auth/me`

用途：
- 注销当前登录用户账户
- 注销是逻辑删除，不会物理删除 MongoDB 文档
- 注销后当前用户状态会变为 `deleted`
- 注销后该用户所有登录身份会变为 `disabled`
- 注销后该用户所有未失效的 refresh token 会被 revoke
- access token 当前不落库；注销后旧 access token 因用户状态已是 `deleted`，无法继续访问受保护接口
- 同一手机号允许重新注册；后端只对 active 的 `AuthIdentity(provider, identifier)` 保持唯一约束

请求头：
- `Authorization: Bearer <access_token>`

请求体：
- 无

成功响应：

```json
{
  "deleted": true
}
```

失败响应：
- `401`: 缺少 access token，access token 非法或已过期
- `404`: User 不存在
- `500`: 删除 User、禁用 AuthIdentity 或 revoke refresh token 失败

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
  "profile": {
    "username": "user8613800138000",
    "gender": "",
    "birthday": "",
    "avatar_url": "https://picpac.oss-cn-shanghai.aliyuncs.com/user-avatar/default.jpg?Expires=1783588103&OSSAccessKeyId=...&Signature=..."
  },
  "status": "created"
}
```

失败响应：
- `401`: 缺少 access token，access token 非法或已过期
- `404`: User 不存在
- `500`: 查询 User 或生成头像访问 URL 失败

### Update My Profile

`PUT /api/v1/me/profile`

用途：
- 更新当前登录用户的 profile
- 后端会接收头像文件并上传到阿里云 OSS，MongoDB 只保存头像 object key，不保存临时 URL
- 如果不上传新的头像文件，会保留当前已有的头像 object key；首次登录创建的默认头像 object key 是 `user-avatar/default.jpg`

请求头：
- `Authorization: Bearer <access_token>`

请求类型：
- `multipart/form-data`

请求字段：
- `username`: string，必填；去首尾空格后长度 1-32
- `gender`: string，必填；枚举值：`male`、`female`、`private`
- `birthday`: string，可选；格式固定为 `YYYY-MM-DD`；传空字符串表示清空生日
- `avatar`: 文件，可选；必须是有效图片文件；不传则保留当前头像

请求示例：
- `username=packmate_user`
- `gender=female`
- `birthday=1998-08-20`
- `avatar=<image file>` 可选

成功响应：

```json
{
  "id": "6821c0c1f1b2f4d5a6b7c8d1",
  "profile": {
    "username": "packmate_user",
    "gender": "female",
    "birthday": "1998-08-20",
    "avatar_url": "https://picpac.oss-cn-shanghai.aliyuncs.com/user-avatar/user_6821c0c1f1b2f4d5a6b7c8d1.png?Expires=1783588103&OSSAccessKeyId=...&Signature=..."
  },
  "status": "created"
}
```

失败响应：
- `400`: 缺少 `username`、`gender`，`username` 超长、`gender` 非法、`birthday` 格式非法，或上传文件不是有效图片
- `401`: 缺少 access token，access token 非法或已过期
- `404`: User 不存在
- `502`: 上传头像到 OSS 失败
- `500`: 更新用户资料或生成头像访问 URL 失败

### List Categories

`GET /api/v1/categories`

用途：
- 获取系统预设 item category 列表
- category 从根目录 `category.json` 初始化到 MongoDB
- 当前版本只有系统分类，不支持用户自定义分类

成功响应：

```json
{
  "categories": [
    {
      "id": "6821c0c1f1b2f4d5a6b7c8a1",
      "key": "document",
      "name": "证件"
    },
    {
      "id": "6821c0c1f1b2f4d5a6b7c8a2",
      "key": "electronics",
      "name": "电子设备"
    }
  ]
}
```

失败响应：
- `500`: 查询 category 列表失败

### Create Item

`POST /api/v1/item`

用途：
- 创建一个用户私有的 item
- 如果上传图片，后端会先上传到阿里云 OSS，MongoDB 只保存图片 object key，不保存临时 URL
- 新创建的 item 会默认写入 `created` 状态
- `user_id` 从当前登录用户读取，不接受前端显式传入

请求类型：
- `multipart/form-data`
- 需要 `Authorization: Bearer <access_token>`

请求字段：
- `name`: string，必填
- `description`: string，可选
- `category_id`: string，可选；不传或传空字符串时后端使用 `key=other` 的 category
- `image`: 文件，可选

成功响应：

```json
{
  "id": "6821c0c1f1b2f4d5a6b7c8d9",
  "user_id": "6821c0c1f1b2f4d5a6b7c8d1",
  "category_id": "6821c0c1f1b2f4d5a6b7c8a1",
  "category_key": "document",
  "category_name": "证件",
  "name": "黑色双肩包",
  "description": "日常出差用",
  "source_image_url": "https://picpac.oss-cn-shanghai.aliyuncs.com/items/item_6821c0c1f1b2f4d5a6b7c8d9/source.jpg?Expires=1783588103&OSSAccessKeyId=...&Signature=...",
  "image_thumbnail_url": "",
  "ai_rendered_image_url": "",
  "status": "created"
}
```

失败响应：
- `400`: 缺少 `name`，`category_id` 非法/不存在，或上传文件不是有效图片
- `401`: access token 缺失、非法或过期
- `502`: 图片上传失败
- `500`: 创建 item、查询默认 category 或生成图片访问 URL 失败

### Batch Create Items

`POST /api/v1/item/batch`

用途：
- 批量创建当前用户的 item
- 适合配合 `POST /api/v1/ai/item-drafts` 使用：AI 先生成草稿，用户确认后批量创建
- `user_id` 从当前登录用户读取，不接受前端显式传入
- 当前最多一次创建 50 个 item
- 批量创建使用 MongoDB transaction，语义上要求全部成功或全部失败

请求类型：
- `application/json`

请求头：
- `Authorization: Bearer <access_token>`

请求字段：
- `items`: array，必填且不能为空
- `items[].name`: string，必填
- `items[].description`: string，可选
- `items[].category_id`: string，可选；不传或传空字符串时后端使用 `key=other` 的 category

请求示例：

```json
{
  "items": [
    {
      "name": "手机",
      "description": "主力机",
      "category_id": "6821c0c1f1b2f4d5a6b7c8a2"
    },
    {
      "name": "充电线",
      "category_id": "6821c0c1f1b2f4d5a6b7c8a2"
    }
  ]
}
```

成功响应：

```json
{
  "items": [
    {
      "id": "6821c0c1f1b2f4d5a6b7c8d9",
      "user_id": "6821c0c1f1b2f4d5a6b7c8d1",
      "category_id": "6821c0c1f1b2f4d5a6b7c8a2",
      "category_key": "electronics",
      "category_name": "电子设备",
      "name": "手机",
      "description": "主力机",
      "source_image_url": "",
      "image_thumbnail_url": "",
      "ai_rendered_image_url": "",
      "status": "created"
    }
  ]
}
```

失败响应：
- `400`: `items` 缺失/为空/超过 50 个，缺少 `items[].name`，或 `category_id` 非法/不存在
- `401`: access token 缺失、非法或过期
- `500`: 批量创建 item、查询默认 category 或生成图片访问 URL 失败

### List Items

`GET /api/v1/item`

用途：
- 查询当前用户的全部 item
- 可通过 `q` 按 item name 或 description 做关键词子串匹配，主要用于中文 item 搜索
- 可通过 `category_id` 按 item category 过滤；可与 `q` 同时使用
- 默认按创建时间倒序返回
- 已逻辑删除的 item 不会出现在列表中

请求参数：
- 需要 `Authorization: Bearer <access_token>`
- `q`: string，可选，按 item name 或 description 子串匹配；当前最大长度为 50 个字符，传空字符串会返回 `400`
- `category_id`: string，可选，按 category 过滤；传空字符串或非法/不存在的 category id 会返回 `400`

搜索示例：

`GET /api/v1/item?q=充电`

分类过滤示例：

`GET /api/v1/item?category_id=6821c0c1f1b2f4d5a6b7c8a2`

组合过滤示例：

`GET /api/v1/item?q=充电&category_id=6821c0c1f1b2f4d5a6b7c8a2`

说明：
- `q=充电` 可以匹配 `手机充电器`、`充电宝` 等名称，也可以匹配 description 中包含 `充电` 的 item
- 中文关键词不做分词，按原始子串匹配
- 英文关键词大小写不敏感
- `q` 搜索 `name` 和 `description`
- `category_id` 过滤当前用户该 category 下的 item

成功响应：

```json
{
  "items": [
    {
      "id": "6821c0c1f1b2f4d5a6b7c8d9",
      "user_id": "6821c0c1f1b2f4d5a6b7c8d1",
      "category_id": "6821c0c1f1b2f4d5a6b7c8a2",
      "category_key": "electronics",
      "category_name": "电子设备",
      "name": "黑色双肩包",
      "description": "日常出差用",
      "source_image_url": "https://picpac.oss-cn-shanghai.aliyuncs.com/items/item_6821c0c1f1b2f4d5a6b7c8d9/source.jpg?Expires=1783588103&OSSAccessKeyId=...&Signature=...",
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
- `400`: `q` 为空/超过最大长度，或 `category_id` 为空/非法/不存在
- `401`: access token 缺失、非法或过期
- `500`: 查询 item 列表或生成图片访问 URL 失败

### Get Item

`GET /api/v1/item/:item_id`

用途：
- 根据 `item_id` 读取单个 item 详情
- 只允许读取当前登录用户自己的 item
- 如果 item 已被逻辑删除，则按不存在处理

路径参数：
- `item_id`: string，必填，item 主键

请求头：
- `Authorization: Bearer <access_token>`

成功响应：

```json
{
  "id": "6821c0c1f1b2f4d5a6b7c8d9",
  "user_id": "6821c0c1f1b2f4d5a6b7c8d1",
  "category_id": "6821c0c1f1b2f4d5a6b7c8a2",
  "category_key": "electronics",
  "category_name": "电子设备",
  "name": "黑色双肩包",
  "description": "日常出差用",
  "source_image_url": "https://picpac.oss-cn-shanghai.aliyuncs.com/items/item_6821c0c1f1b2f4d5a6b7c8d9/source.jpg?Expires=1783588103&OSSAccessKeyId=...&Signature=...",
  "image_thumbnail_url": "",
  "ai_rendered_image_url": "",
  "status": "created"
}
```

失败响应：
- `400`: 缺少 `item_id`，或 `item_id` 不是合法 ObjectID
- `401`: access token 缺失、非法或过期
- `404`: item 不存在
- `500`: 查询 item 或生成图片访问 URL 失败

### Update Item

`PUT /api/v1/item/:item_id`

用途：
- 更新单个 item 的名称、描述和可选图片
- 只允许更新当前登录用户自己的 item
- 如果上传新图片，会覆盖后端保存的 source image object key；响应里的 `source_image_url` 会返回新的临时 signed URL
- 如果 item 已被逻辑删除，则不允许更新

请求类型：
- `multipart/form-data`

路径参数：
- `item_id`: string，必填，item 主键

请求头：
- `Authorization: Bearer <access_token>`

请求字段：
- `name`: string，必填
- `description`: string，可选
- `category_id`: string，可选；不传时保留原 category，传空字符串时后端使用 `key=other` 的 category
- `image`: 文件，可选

成功响应：

```json
{
  "id": "6821c0c1f1b2f4d5a6b7c8d9",
  "user_id": "6821c0c1f1b2f4d5a6b7c8d1",
  "category_id": "6821c0c1f1b2f4d5a6b7c8a2",
  "category_key": "electronics",
  "category_name": "电子设备",
  "name": "黑色双肩包升级版",
  "description": "更新后的描述",
  "source_image_url": "https://picpac.oss-cn-shanghai.aliyuncs.com/items/item_6821c0c1f1b2f4d5a6b7c8d9/source.png?Expires=1783588103&OSSAccessKeyId=...&Signature=...",
  "image_thumbnail_url": "",
  "ai_rendered_image_url": "",
  "status": "created"
}
```

失败响应：
- `400`: 缺少 `name`，`item_id` 非法，`category_id` 非法/不存在，或上传文件不是有效图片
- `401`: access token 缺失、非法或过期
- `404`: item 不存在
- `502`: 图片上传失败
- `500`: 更新 item 或生成图片访问 URL 失败

### Delete Item

`DELETE /api/v1/item/:item_id`

用途：
- 逻辑删除单个 item
- 只允许删除当前登录用户自己的 item
- 删除后会把 `status` 置为 `deleted`，不会真的从 MongoDB 中移除

路径参数：
- `item_id`: string，必填，item 主键

请求头：
- `Authorization: Bearer <access_token>`

成功响应：

```json
{
  "deleted": true
}
```

失败响应：
- `400`: 缺少 `item_id`，或 `item_id` 不是合法 ObjectID
- `401`: access token 缺失、非法或过期
- `404`: item 不存在
- `500`: 删除 item 失败

### Recommend Pack Items

`POST /api/v1/ai/pack/item-recommendations`

用途：
- 根据用户输入的 pack name 和可选 description，推荐当前用户已有 item 中适合加入该 pack 的 item
- 此接口只做推荐，不会创建 pack、不会创建 item、不会修改 pack/checklist/item
- 后端会读取当前用户的 item 列表并交给 DeepSeek 做语义推荐
- 当前最多返回 15 个推荐 item；前端可在收到结果后与本地已勾选状态合并，再询问用户是否一键添加

请求类型：
- `application/json`

请求头：
- `Authorization: Bearer <access_token>`

请求字段：
- `pack_name`: string，必填；去首尾空格后长度 1-64
- `description`: string，可选；去首尾空格后最大长度 200

请求示例：

```json
{
  "pack_name": "日本出差",
  "description": "东京 5 天商务行程"
}
```

成功响应：

```json
{
  "recommended_items": [
    {
      "id": "6821c0c1f1b2f4d5a6b7c8d9",
      "name": "护照"
    },
    {
      "id": "6821c0c1f1b2f4d5a6b7c8da",
      "name": "充电器"
    }
  ]
}
```

空推荐响应：

```json
{
  "recommended_items": []
}
```

说明：
- 返回值只包含推荐 item 的 `id` 和 `name`，不返回 category、图片 URL 或完整 item 详情
- 推荐结果只会包含当前用户已有且未逻辑删除的 item
- AI 返回结果会在后端做 ref 校验、去重和最多 15 个的截断兜底

失败响应：
- `400`: 缺少 `pack_name`，`pack_name` 超长，或 `description` 超长
- `401`: access token 缺失、非法或过期
- `500`: 查询 item/category 列表失败，或 DeepSeek 推荐失败

### Generate Item Drafts

`POST /api/v1/ai/item-drafts`

用途：
- 从用户自然语言中提取待创建 item 草稿
- 此接口只生成草稿，不会创建 item，不会修改用户物品库
- 后端会读取系统 category 列表，并让 AI 为每个 draft 选择 category
- AI 只返回 category key；后端根据真实 category 表映射出 `category_id/category_name`
- 如果 AI 返回未知 category key，后端会 fallback 到 `key=other` 的 category
- 当前最多返回 50 个 draft item；同一次结果内重复 name 会被去重

请求类型：
- `application/json`

请求头：
- `Authorization: Bearer <access_token>`

请求字段：
- `text`: string，必填；用户自然语言输入，去首尾空格后长度 1-500

请求示例：

```json
{
  "text": "请帮我添加手机，充电线，相机"
}
```

成功响应：

```json
{
  "draft_items": [
    {
      "name": "手机",
      "category_id": "6821c0c1f1b2f4d5a6b7c8a2",
      "category_key": "electronics",
      "category_name": "电子设备"
    },
    {
      "name": "充电线",
      "category_id": "6821c0c1f1b2f4d5a6b7c8a2",
      "category_key": "electronics",
      "category_name": "电子设备"
    }
  ]
}
```

空草稿响应：

```json
{
  "draft_items": []
}
```

说明：
- 此接口用于前端展示待创建草稿，由用户确认/编辑后再调用批量创建接口
- 第一版只做本次 AI draft 内去重，不检查用户物品库中是否已有同名 item
- 不支持上传图片；图片仍需通过单个 item 更新/创建流程处理

失败响应：
- `400`: 缺少 `text`，或 `text` 超长
- `401`: access token 缺失、非法或过期
- `500`: 查询 category 列表失败，或 DeepSeek 草稿提取失败

### Create Pack

`POST /api/v1/pack`

用途：
- 创建一个用户的 pack，用于规划一次打包清单
- `user_id` 从当前登录用户读取，不接受前端显式传入
- `items` 中的每个 item 都必须存在、未删除且属于当前登录用户
- 新创建的 pack 会默认写入 `created` 状态

请求类型：
- `application/json`
- 需要 `Authorization: Bearer <access_token>`

请求字段：
- `name`: string，必填
- `description`: string，可选
- `items`: string array，可选，item id 列表

请求示例：

```json
{
  "name": "日本出差",
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
- `400`: 缺少 `name`，或 `items` 中存在非法 item id
- `401`: access token 缺失、非法或过期
- `404`: `items` 中存在不属于当前用户、已删除或不存在的 item
- `500`: 创建 pack 失败

### List Packs

`GET /api/v1/pack`

用途：
- 查询 pack 列表
- 可通过 `q` 按 pack name 或 description 做关键词子串匹配，主要用于中文 pack 搜索
- `status` 是内部状态，不支持作为 query 参数过滤
- 默认按创建时间倒序返回
- 已逻辑删除的 pack 不会出现在列表中

请求参数：
- 需要 `Authorization: Bearer <access_token>`
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
- `400`: `q` 为空/超过最大长度
- `401`: access token 缺失、非法或过期
- `500`: 查询 pack 列表失败

### Get Pack

`GET /api/v1/pack/:pack_id`

用途：
- 根据 `pack_id` 读取单个 pack 详情
- 只允许读取当前登录用户自己的 pack
- 如果 pack 已被逻辑删除，则按不存在处理
- `status` 是内部状态，不支持作为 query 参数过滤

路径参数：
- `pack_id`: string，必填，pack 主键

请求头：
- `Authorization: Bearer <access_token>`

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
- `401`: access token 缺失、非法或过期
- `404`: pack 不存在
- `500`: 查询 pack 失败

### Update Pack Profile

`PATCH /api/v1/pack/:pack_id/profile`

用途：
- 更新单个 pack 的基本信息
- 只允许更新当前登录用户自己的 pack
- 只更新 `name`、`description`，不会修改 pack 内 item 列表
- `name` 必填
- `description` 传空字符串表示清空描述
- 后端会保留 `id`、`user_id`、`items`、`status`、`created_at` 等字段，并更新 `updated_at`
- 如果 pack 已被逻辑删除，则不允许更新

请求类型：
- `application/json`

路径参数：
- `pack_id`: string，必填，pack 主键

请求头：
- `Authorization: Bearer <access_token>`

请求字段：
- `name`: string，必填
- `description`: string，可选

请求示例：

```json
{
  "name": "日本出差升级版",
  "description": "东京 6 天商务行程"
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
- `400`: 缺少 `name`，或 `pack_id` 非法
- `401`: access token 缺失、非法或过期
- `404`: pack 不存在
- `500`: 更新 pack 失败

### Add Pack Items

`POST /api/v1/pack/:pack_id/items`

用途：
- 批量添加 item 到 pack
- 只允许更新当前登录用户自己的 pack
- `items` 必填且不能为空
- `items` 中的每个 item 都必须存在、未删除且属于当前登录用户
- 已经在 pack 里的 item 会被忽略，后端会保证 pack 内 item id 不重复
- 如果 pack 已被逻辑删除，则不允许更新

请求类型：
- `application/json`

路径参数：
- `pack_id`: string，必填，pack 主键

请求头：
- `Authorization: Bearer <access_token>`

请求字段：
- `items`: string array，必填，待添加的 item id 列表

请求示例：

```json
{
  "items": [
    "6821c0c1f1b2f4d5a6b7c8d9",
    "6821c0c1f1b2f4d5a6b7c8da"
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
    "6821c0c1f1b2f4d5a6b7c8d9",
    "6821c0c1f1b2f4d5a6b7c8da"
  ],
  "status": "created"
}
```

失败响应：
- `400`: 缺少 `items`，`items` 为空，`pack_id` 非法，或 `items` 中存在非法 item id
- `401`: access token 缺失、非法或过期
- `404`: pack 不存在，或 `items` 中存在不属于当前用户、已删除或不存在的 item
- `500`: 更新 pack 失败

### Remove Pack Items

`DELETE /api/v1/pack/:pack_id/items`

用途：
- 批量从 pack 删除 item
- 只允许更新当前登录用户自己的 pack
- `items` 必填且不能为空
- 不在 pack 里的 item 会被忽略，重复调用结果一致
- 如果 pack 已被逻辑删除，则不允许更新

请求类型：
- `application/json`

路径参数：
- `pack_id`: string，必填，pack 主键

请求头：
- `Authorization: Bearer <access_token>`

请求字段：
- `items`: string array，必填，待删除的 item id 列表

请求示例：

```json
{
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
    "6821c0c1f1b2f4d5a6b7c8da"
  ],
  "status": "created"
}
```

失败响应：
- `400`: 缺少 `items`，`items` 为空，`pack_id` 非法，或 `items` 中存在非法 item id
- `401`: access token 缺失、非法或过期
- `404`: pack 不存在
- `500`: 更新 pack 失败

### Delete Pack

`DELETE /api/v1/pack/:pack_id`

用途：
- 逻辑删除单个 pack
- 只允许删除当前登录用户自己的 pack
- 删除后会把 `status` 置为 `deleted`，不会真的从 MongoDB 中移除

路径参数：
- `pack_id`: string，必填，pack 主键

请求头：
- `Authorization: Bearer <access_token>`

成功响应：

```json
{
  "deleted": true
}
```

失败响应：
- `400`: 缺少 `pack_id`，或 `pack_id` 不是合法 ObjectID
- `401`: access token 缺失、非法或过期
- `404`: pack 不存在或已被逻辑删除
- `500`: 删除 pack 失败

### Create Checklist

`POST /api/v1/checklist`

用途：
- 创建一个 checklist
- `user_id` 从当前登录用户读取，不接受前端显式传入
- 新创建的 checklist 会默认写入 `created` 状态
- `items` 是 line item 列表；如果 `reference_type` 是 `item`，`reference_id` 必须是 item id，且不能传 `snapshot`；如果 `reference_type` 是 `snapshot`，`reference_id` 必须为空，且必须传 `snapshot.name`
- `reference_type` 是 `item` 时，后端会校验对应 item 存在、未被逻辑删除且属于当前登录用户
- line item 初始状态统一为 `unchecked`
- 当前正式接口没有 `pack_id` 输入；如果前端是先从 pack 展开成 line items 再调用该接口，本次后端能校验的是展开后的 item owner，而不是原始 pack id

请求类型：
- `application/json`
- 需要 `Authorization: Bearer <access_token>`

请求字段：
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
- `400`: 缺少 `name`、`target_date`，字段格式非法，或 line item 非法
- `401`: access token 缺失、非法或过期
- `404`: 引用的 item 不存在、已删除或不属于当前用户
- `500`: 创建 checklist 失败

### List Checklists

`GET /api/v1/checklist`

用途：
- 查询 checklist 列表
- 可通过 `q` 按 checklist name 或 description 做关键词子串匹配
- 默认按创建时间倒序返回
- 已逻辑删除的 checklist 不会出现在列表中

请求参数：
- 需要 `Authorization: Bearer <access_token>`
- `q`: string，可选，按 checklist name 或 description 子串匹配；当前最大长度为 50 个字符，传空字符串会返回 `400`

成功响应：

```json
{
  "checklists": []
}
```

失败响应：
- `400`: `q` 为空/超过最大长度
- `401`: access token 缺失、非法或过期
- `500`: 查询 checklist 列表失败

### Get Checklist

`GET /api/v1/checklist/:checklist_id`

用途：
- 根据 `checklist_id` 读取单个 checklist 详情
- 只允许读取当前登录用户自己的 checklist
- 如果 checklist 已被逻辑删除，则按不存在处理

路径参数：
- `checklist_id`: string，必填，checklist 主键

请求头：
- `Authorization: Bearer <access_token>`

成功响应：
- 同 Create Checklist 成功响应结构

失败响应：
- `400`: 缺少 `checklist_id`，或 `checklist_id` 不是合法 ObjectID
- `401`: access token 缺失、非法或过期
- `404`: checklist 不存在
- `500`: 查询 checklist 失败

### Update Checklist

`PUT /api/v1/checklist/:checklist_id`

用途：
- 更新单个 checklist 的 metadata
- 只允许更新当前登录用户自己的 checklist
- 前端提交更新后的 `name`、`description`、`target_date`
- 后端会保留 `id`、`user_id`、`status`、`created_at` 等系统字段，并更新 `updated_at`
- `items` 不允许通过该接口更新；line item 需要使用 Add/Remove Checklist Line Items 接口修改
- 如果 checklist 已被逻辑删除，则不允许更新

请求类型：
- `application/json`

路径参数：
- `checklist_id`: string，必填，checklist 主键

请求头：
- `Authorization: Bearer <access_token>`

请求字段：
- `name`: string，必填
- `description`: string，可选
- `target_date`: string，必填，格式为 `YYYY-MM-DD`

失败响应：
- `400`: 缺少 `name`、`target_date`，`checklist_id` 非法，字段格式非法，或请求体包含 `items`
- `401`: access token 缺失、非法或过期
- `404`: checklist 不存在
- `500`: 更新 checklist 失败

### Add Checklist Line Items

`POST /api/v1/checklist/:checklist_id/items`

用途：
- 向指定 checklist 批量增加 line item
- 只允许更新当前登录用户自己的 checklist
- 新增 line item 会自动生成自己的 `id`
- 新增 line item 初始状态统一为 `unchecked`
- 如果 line item 的 `reference_type` 是 `item`，后端会校验对应 item 存在、未被逻辑删除且属于当前登录用户
- 更新成功后会更新 checklist 的 `updated_at`

请求类型：
- `application/json`

路径参数：
- `checklist_id`: string，必填，checklist 主键

请求头：
- `Authorization: Bearer <access_token>`

请求字段：
- `items`: object array，必填且不能为空
- `items[].reference_type`: string，必填，只支持 `item` 或 `snapshot`
- `items[].reference_id`: string。当 `reference_type` 为 `item` 时必填且必须是存在、未删除且属于当前登录用户的 item id；当 `reference_type` 为 `snapshot` 时必须为空
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
- `400`: 缺少 `items`，`checklist_id` 非法，或 line item 非法
- `401`: access token 缺失、非法或过期
- `404`: checklist 不存在，或引用的 item 不存在、已删除或不属于当前用户
- `500`: 更新 checklist 失败

### Remove Checklist Line Items

`DELETE /api/v1/checklist/:checklist_id/items`

用途：
- 从指定 checklist 批量移除 line item
- 只允许更新当前登录用户自己的 checklist
- 只有请求中的所有 `line_item_ids` 都属于当前 checklist 时才会更新
- 更新成功后会更新 checklist 的 `updated_at`

请求类型：
- `application/json`

路径参数：
- `checklist_id`: string，必填，checklist 主键

请求头：
- `Authorization: Bearer <access_token>`

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
- `401`: access token 缺失、非法或过期
- `404`: checklist 不存在，或存在不属于该 checklist 的 line item id
- `500`: 更新 checklist 失败

### Update Checklist Line Item Status

`PATCH /api/v1/checklist/:checklist_id/items/:line_item_id/status`

用途：
- 更新指定 checklist 中单个 line item 的勾选状态
- 每次请求只更新一个 line item 的 `status`
- 只允许更新当前登录用户自己的 checklist
- 更新成功后会更新 checklist 的 `updated_at`
- 这里不需要再额外校验 line item 引用的 item owner；因为对外暴露的是 checklist 内部 line item 状态变更，owner 边界由 checklist 本身和 create/add 时的 item owner 校验保证

请求类型：
- `application/json`

路径参数：
- `checklist_id`: string，必填，checklist 主键
- `line_item_id`: string，必填，checklist line item 主键

请求头：
- `Authorization: Bearer <access_token>`

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
- `401`: access token 缺失、非法或过期
- `404`: checklist 不存在，或 line item 不属于该 checklist
- `500`: 更新 checklist line item status 失败，或读取更新后的 checklist 失败

### Delete Checklist

`DELETE /api/v1/checklist/:checklist_id`

用途：
- 逻辑删除单个 checklist
- 只允许删除当前登录用户自己的 checklist
- 删除后会把 `status` 置为 `deleted`，不会真的从 MongoDB 中移除

路径参数：
- `checklist_id`: string，必填，checklist 主键

请求头：
- `Authorization: Bearer <access_token>`

成功响应：

```json
{
  "deleted": true
}
```

失败响应：
- `400`: 缺少 `checklist_id`，或 `checklist_id` 不是合法 ObjectID
- `401`: access token 缺失、非法或过期
- `404`: checklist 不存在或已被逻辑删除
- `500`: 删除 checklist 失败

## Planned Domain APIs

后续仍计划补充以下正式接口：
- User authentication
