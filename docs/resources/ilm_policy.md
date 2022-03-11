---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "minio_ilm_policy Resource - terraform-provider-minio"
subcategory: ""
description: |-
  minio_ilm_policy handles lifecycle settings for a given minio_s3_bucket.
---

# minio_ilm_policy (Resource)

`minio_ilm_policy` handles lifecycle settings for a given `minio_s3_bucket`.

## Example Usage

```terraform
resource "minio_s3_bucket" "bucket" {
  bucket = "bucket"
}

resource "minio_ilm_policy" "bucket-lifecycle-rules" {
  bucket = minio_s3_bucket.bucket.bucket

  rule {
    id         = "expire-7d"
    expiration = 7
  }
}
```

## Schema

### Required

- **bucket** (String)
- **rule** (Block List, Min: 1) (see [below for nested schema](#nested-schema-for-rule))

### Optional

- **id** (String) The ID of this resource.

### Nested Schema for `rule`

Required:

- **id** (String) The ID of this resource.

Optional:

- **expiration** (String)
- **filter** (String)

Read-Only:

- **status** (String)

