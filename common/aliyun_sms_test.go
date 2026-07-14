package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAliyunSmsConfigured(t *testing.T) {
	oldID, oldSecret, oldSign, oldTpl := AliyunSmsAccessKeyId, AliyunSmsAccessKeySecret, AliyunSmsSignName, AliyunSmsTemplateCode
	t.Cleanup(func() {
		AliyunSmsAccessKeyId, AliyunSmsAccessKeySecret, AliyunSmsSignName, AliyunSmsTemplateCode = oldID, oldSecret, oldSign, oldTpl
	})
	AliyunSmsAccessKeyId, AliyunSmsAccessKeySecret, AliyunSmsSignName, AliyunSmsTemplateCode = "", "", "", ""
	assert.False(t, AliyunSmsConfigured())
	AliyunSmsAccessKeyId, AliyunSmsAccessKeySecret, AliyunSmsSignName, AliyunSmsTemplateCode = "id", "sec", "sign", "tpl"
	assert.True(t, AliyunSmsConfigured())
}
