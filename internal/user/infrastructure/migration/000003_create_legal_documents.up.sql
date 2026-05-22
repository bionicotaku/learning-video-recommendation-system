create table if not exists app_user.legal_documents (
  document_type text primary key
    check (document_type in ('privacy-policy', 'user-agreement')),
  title text not null,
  markdown text not null,
  version text,
  updated_at timestamptz
);

insert into app_user.legal_documents (
  document_type,
  title,
  markdown,
  version,
  updated_at
) values
  (
    'privacy-policy',
    '隐私政策',
    $markdown$# 隐私政策

## 我们收集的信息

我们会在你使用学习功能时处理必要的账户信息、学习记录和设备环境信息，用于提供课程推荐、学习进度和账号安全能力。

## 信息使用方式

- 展示你的学习统计和连续学习记录
- 改善视频推荐与练习体验
- 保护账号与服务安全

## 联系我们

如需了解、导出或删除个人信息，请通过应用内反馈入口联系我们。
$markdown$,
    '2026-05-21',
    '2026-05-21T00:00:00Z'::timestamptz
  ),
  (
    'user-agreement',
    '用户协议',
    $markdown$# 用户协议

## 服务说明

本应用提供基于视频内容的语言学习体验，包括字幕学习、单词解释、练习题和学习进度记录。

## 使用规则

- 请勿上传或传播侵权、违法或不适合学习场景的内容
- 请妥善保管你的账号登录状态
- 请尊重内容版权和其他用户权益

## 协议更新

我们可能根据产品功能和法律要求更新本协议。更新后会在应用内展示最新版本。
$markdown$,
    '2026-05-21',
    '2026-05-21T00:00:00Z'::timestamptz
  )
on conflict (document_type) do update
set
  title = excluded.title,
  markdown = excluded.markdown,
  version = excluded.version,
  updated_at = excluded.updated_at;
