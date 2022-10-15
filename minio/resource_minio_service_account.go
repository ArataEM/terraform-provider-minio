package minio

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/minio/madmin-go"
)

func resourceMinioServiceAccount() *schema.Resource {
	return &schema.Resource{
		CreateContext: minioCreateServiceAccount,
		ReadContext:   minioReadServiceAccount,
		UpdateContext: minioUpdateServiceAccount,
		DeleteContext: minioDeleteServiceAccount,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"target_user": {
				Type:     schema.TypeString,
				Required: true,
			},
			"disable_user": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Disable service account",
			},
			"update_secret": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "rotate secret key",
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"secret_key": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"access_key": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func minioCreateServiceAccount(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	serviceAccountConfig := ServiceAccountConfig(d, meta)

	var err error
	targetUser := serviceAccountConfig.MinioTargetUser

	serviceAccount, err := serviceAccountConfig.MinioAdmin.AddServiceAccount(ctx, madmin.AddServiceAccountReq{
		Policy:     nil,
		TargetUser: targetUser,
	})
	if err != nil {
		return NewResourceError("error creating service account", targetUser, err)
	}
	accessKey := serviceAccount.AccessKey
	secretKey := serviceAccount.SecretKey

	d.SetId(aws.StringValue(&accessKey))
	_ = d.Set("access_key", accessKey)
	_ = d.Set("secret_key", secretKey)

	if serviceAccountConfig.MinioDisableUser {
		err = serviceAccountConfig.MinioAdmin.UpdateServiceAccount(ctx, accessKey, madmin.UpdateServiceAccountReq{NewStatus: "off"})
		if err != nil {
			return NewResourceError("error disabling service account %s: %s", d.Id(), err)
		}
	}

	return minioReadServiceAccount(ctx, d, meta)
}

func minioUpdateServiceAccount(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	serviceAccountConfig := ServiceAccountConfig(d, meta)

	var err error
	secretKey := ""

	if serviceAccountConfig.MinioUpdateKey {
		if secretKey, err = generateSecretAccessKey(); err != nil {
			return NewResourceError("error creating service account", d.Id(), err)
		}
	}

	if d.HasChange(serviceAccountConfig.MinioTargetUser) {
		on, nn := d.GetChange(serviceAccountConfig.MinioTargetUser)

		log.Println("[DEBUG] Update service account:", serviceAccountConfig.MinioTargetUser)
		err := serviceAccountConfig.MinioAdmin.DeleteServiceAccount(ctx, on.(string))
		if err != nil {
			return NewResourceError("error updating service account %s: %s", d.Id(), err)
		}

		_, err = serviceAccountConfig.MinioAdmin.AddServiceAccount(ctx, madmin.AddServiceAccountReq{AccessKey: nn.(string), SecretKey: secretKey})
		if err != nil {
			return NewResourceError("error updating service account %s: %s", d.Id(), err)
		}

		d.SetId(nn.(string))
	}

	serviceAccountStatus := ServiceAccountStatus{
		AccessKey:     d.Id(),
		SecretKey:     secretKey,
		AccountStatus: "on",
	}

	if serviceAccountConfig.MinioDisableUser {
		serviceAccountStatus.AccountStatus = "off"
	}

	serviceAccountServerInfo, _ := serviceAccountConfig.MinioAdmin.InfoServiceAccount(ctx, serviceAccountConfig.MinioTargetUser)
	if serviceAccountServerInfo.AccountStatus != serviceAccountStatus.AccountStatus {
		err := serviceAccountConfig.MinioAdmin.UpdateServiceAccount(ctx, serviceAccountStatus.AccessKey, madmin.UpdateServiceAccountReq{
			NewStatus: serviceAccountStatus.AccountStatus,
		})
		if err != nil {
			return NewResourceError("error to disable service account %s: %s", d.Id(), err)
		}
	}

	if serviceAccountConfig.MinioUpdateKey {
		err := serviceAccountConfig.MinioAdmin.UpdateServiceAccount(ctx, serviceAccountStatus.AccessKey, madmin.UpdateServiceAccountReq{
			NewPolicy:    nil,
			NewSecretKey: serviceAccountStatus.SecretKey,
		})
		if err != nil {
			return NewResourceError("error updating service account Key %s: %s", d.Id(), err)
		}

		_ = d.Set("secret_key", secretKey)
	}

	return minioReadServiceAccount(ctx, d, meta)
}

func minioReadServiceAccount(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	serviceAccountConfig := ServiceAccountConfig(d, meta)

	output, err := serviceAccountConfig.MinioAdmin.InfoServiceAccount(ctx, d.Id())
	if err != nil {
		return NewResourceError("error reading service account %s: %s", d.Id(), err)
	}

	log.Printf("[WARN] (%v)", output)

	if _, ok := d.GetOk("access_key"); !ok {
		_ = d.Set("access_key", d.Id())
	}

	if err := d.Set("status", output.AccountStatus); err != nil {
		return NewResourceError("reading service account failed", d.Id(), err)
	}

	return nil
}

func minioDeleteServiceAccount(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	serviceAccountConfig := ServiceAccountConfig(d, meta)

	err := deleteMinioServiceAccount(ctx, serviceAccountConfig)
	if err != nil {
		return NewResourceError("error deleting service account %s: %s", d.Id(), err)
	}

	// Actively set resource as deleted
	d.SetId("")

	return nil
}

func deleteMinioServiceAccount(ctx context.Context, serviceAccountConfig *S3MinioServiceAccountConfig) error {
	log.Println("[DEBUG] Deleting service account request:", serviceAccountConfig.MinioAccessKey)
	err := serviceAccountConfig.MinioAdmin.DeleteServiceAccount(ctx, serviceAccountConfig.MinioAccessKey)
	if err != nil {
		serviceAccountList, err := serviceAccountConfig.MinioAdmin.ListServiceAccounts(ctx, serviceAccountConfig.MinioTargetUser)
		if err != nil {
			return err
		}
		if !Contains(serviceAccountList.Accounts, serviceAccountConfig.MinioAccessKey) {
			return nil
		}

		return err
	}
	return nil
}
