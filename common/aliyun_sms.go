package common

import (
	"errors"
	"fmt"
	"strings"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v4/client"
	"github.com/alibabacloud-go/tea/tea"
)

func AliyunSmsConfigured() bool {
	return strings.TrimSpace(AliyunSmsAccessKeyId) != "" &&
		strings.TrimSpace(AliyunSmsAccessKeySecret) != "" &&
		strings.TrimSpace(AliyunSmsSignName) != "" &&
		strings.TrimSpace(AliyunSmsTemplateCode) != ""
}

func SendAliyunSms(phone, code string) error {
	if !AliyunSmsConfigured() {
		return errors.New("sms service not configured")
	}

	normalizedPhone, err := NormalizePhone(phone)
	if err != nil {
		return err
	}

	codeKey := strings.TrimSpace(AliyunSmsTemplateParamCodeKey)
	if codeKey == "" {
		codeKey = "code"
	}

	templateParamBytes, err := Marshal(map[string]string{codeKey: code})
	if err != nil {
		return err
	}

	config := &openapi.Config{
		AccessKeyId:     tea.String(strings.TrimSpace(AliyunSmsAccessKeyId)),
		AccessKeySecret: tea.String(strings.TrimSpace(AliyunSmsAccessKeySecret)),
	}
	if endpoint := strings.TrimSpace(AliyunSmsEndpoint); endpoint != "" {
		config.Endpoint = tea.String(endpoint)
	}

	smsClient, err := dysmsapi.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create sms client: %w", err)
	}

	request := &dysmsapi.SendSmsRequest{
		PhoneNumbers:  tea.String(normalizedPhone),
		SignName:      tea.String(strings.TrimSpace(AliyunSmsSignName)),
		TemplateCode:  tea.String(strings.TrimSpace(AliyunSmsTemplateCode)),
		TemplateParam: tea.String(string(templateParamBytes)),
	}

	response, err := smsClient.SendSms(request)
	if err != nil {
		logAliyunSmsFailure(normalizedPhone, "", err.Error())
		return fmt.Errorf("failed to send sms: %s", err.Error())
	}

	if response == nil || response.Body == nil {
		logAliyunSmsFailure(normalizedPhone, "", "empty response")
		return errors.New("failed to send sms: empty response")
	}

	body := response.Body
	if body.Code == nil || tea.StringValue(body.Code) != "OK" {
		requestId := ""
		if body.RequestId != nil {
			requestId = tea.StringValue(body.RequestId)
		}
		msg := "unknown error"
		if body.Message != nil {
			msg = tea.StringValue(body.Message)
		}
		codeVal := ""
		if body.Code != nil {
			codeVal = tea.StringValue(body.Code)
		}
		logAliyunSmsFailure(normalizedPhone, requestId, fmt.Sprintf("code=%s, message=%s", codeVal, msg))
		return fmt.Errorf("failed to send sms: %s", msg)
	}

	return nil
}

func logAliyunSmsFailure(phone, requestId, detail string) {
	SysLog(fmt.Sprintf("aliyun sms send failed: requestId=%s phone=%s detail=%s", requestId, MaskPhone(phone), detail))
}
