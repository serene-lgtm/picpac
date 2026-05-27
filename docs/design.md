# picpac设计文档

## 概述
Picpac是一款个人物品管理类型的手机端app，旨在提供一个互联网sandbox帮助经常出行的朋友们先在手机上打包好行李，以便在实际打包的时候有条不紊地check done。

## 核心概念
### Item – 物品资产
经过提炼、可复用的标准品定义，是被引用的源头
用户愿意主动、严谨地去维护
持久存在，长期维护
具有唯一性（MVP version暂不设置去重标准）

### Item Snapshot – 物品快照
为特定清单服务的物品实例，随意调整互不干扰
不需要维护
不具有唯一性
物品快照可转换为物品资产

### Pack – 物品套组
按某种场景/偏好/习惯/类型整理好的标准化套组，例如高原必备套组，防暑降温必备套组，会长期维护，只包含物品资产。
Pack中的item来源仅支持从my items中导入。

### Checklist – 每次出行的行李清单
为出行设定的待查清单，可包含物品资产和物品快照。
Checklist中的item来源：
从my items中导入一个或多个
从pack中导入一个或多个或整组
create一个或多个物品快照

### Line Item - checklist中的物品条例
可以是Item也可以是Item Snapshot，有status用来显示是否checked。

## MVP版本需求
User account的注册与登录，支持手机号与微信登录
Item的增删改查
Pack的增删改查
Checklist的增删改查
