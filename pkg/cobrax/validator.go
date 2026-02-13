package cobrax

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// ==================== RequiredValidator ====================

func (v *RequiredValidator) Validate(value any) error {
	if value == nil {
		return errors.New(v.getMessage())
	}

	switch val := value.(type) {
	case string:
		if strings.TrimSpace(val) == "" {
			return errors.New(v.getMessage())
		}
	case []string:
		if len(val) == 0 {
			return errors.New(v.getMessage())
		}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		// 数值类型零值也视为有效值
	case float32, float64:
		// 浮点数类型零值也视为有效值
	case bool:
		// 布尔类型任何值都视为有效
	case time.Time:
		if val.IsZero() {
			return errors.New(v.getMessage())
		}
	default:
		if reflect.ValueOf(val).IsZero() {
			return errors.New(v.getMessage())
		}
	}
	return nil
}

func (v *RequiredValidator) getMessage() string {
	if v.Message != "" {
		return v.Message
	}
	return "参数不能为空"
}

// ==================== MinLengthValidator ====================

func (v *MinLengthValidator) Validate(value any) error {
	str, ok := value.(string)
	if !ok {
		return errors.New("MinLengthValidator 只能验证字符串类型")
	}

	if len(str) < v.Min {
		if v.Message != "" {
			return errors.New(v.Message)
		}
		return fmt.Errorf("参数长度不能少于%d个字符", v.Min)
	}
	return nil
}

// ==================== MaxLengthValidator ====================

func (v *MaxLengthValidator) Validate(value any) error {
	str, ok := value.(string)
	if !ok {
		return errors.New("MaxLengthValidator 只能验证字符串类型")
	}

	if len(str) > v.Max {
		if v.Message != "" {
			return errors.New(v.Message)
		}
		return fmt.Errorf("参数长度不能超过%d个字符", v.Max)
	}
	return nil
}

// ==================== RegexValidator ====================

func (v *RegexValidator) Validate(value any) error {
	str, ok := value.(string)
	if !ok {
		return errors.New("RegexValidator 只能验证字符串类型")
	}

	matched, err := regexp.MatchString(v.Pattern, str)
	if err != nil {
		return fmt.Errorf("正则表达式错误: %v", err)
	}

	if !matched {
		if v.Message != "" {
			return errors.New(v.Message)
		}
		return errors.New("参数格式不正确")
	}
	return nil
}

// ==================== MinValueValidator ====================

func (v *MinValueValidator) Validate(value any) error {
	switch val := value.(type) {
	case int:
		min, ok := v.Min.(int)
		if !ok {
			return errors.New("MinValueValidator: Min 值类型不匹配")
		}
		if val < min {
			return v.getErrorMessage(min)
		}
	case int64:
		min, ok := v.Min.(int64)
		if !ok {
			return errors.New("MinValueValidator: Min 值类型不匹配")
		}
		if val < min {
			return v.getErrorMessage(min)
		}
	case float64:
		min, ok := v.Min.(float64)
		if !ok {
			return errors.New("MinValueValidator: Min 值类型不匹配")
		}
		if val < min {
			return v.getErrorMessage(min)
		}
	default:
		return errors.New("MinValueValidator 只能验证数值类型")
	}
	return nil
}

func (v *MinValueValidator) getErrorMessage(min any) error {
	if v.Message != "" {
		return errors.New(v.Message)
	}
	return fmt.Errorf("参数值不能小于%v", min)
}

// ==================== MaxValueValidator ====================

func (v *MaxValueValidator) Validate(value any) error {
	switch val := value.(type) {
	case int:
		max, ok := v.Max.(int)
		if !ok {
			return errors.New("MaxValueValidator: Max 值类型不匹配")
		}
		if val > max {
			return v.getErrorMessage(max)
		}
	case int64:
		max, ok := v.Max.(int64)
		if !ok {
			return errors.New("MaxValueValidator: Max 值类型不匹配")
		}
		if val > max {
			return v.getErrorMessage(max)
		}
	case float64:
		max, ok := v.Max.(float64)
		if !ok {
			return errors.New("MaxValueValidator: Max 值类型不匹配")
		}
		if val > max {
			return v.getErrorMessage(max)
		}
	default:
		return errors.New("MaxValueValidator 只能验证数值类型")
	}
	return nil
}

func (v *MaxValueValidator) getErrorMessage(max any) error {
	if v.Message != "" {
		return errors.New(v.Message)
	}
	return fmt.Errorf("参数值不能大于%v", max)
}

// ==================== Command 校验方法 ====================

// ValidateFlags 验证命令的所有标志
func (c *Command) ValidateFlags() error {
	if len(c.validators) == 0 {
		return nil
	}

	for flagName, validators := range c.validators {
		flag := c.Command.Flags().Lookup(flagName)
		if flag == nil {
			continue
		}

		var value any
		var err error

		switch flag.Value.Type() {
		case "string":
			value, err = c.Command.Flags().GetString(flagName)
		case "int":
			value, err = c.Command.Flags().GetInt(flagName)
		case "bool":
			value, err = c.Command.Flags().GetBool(flagName)
		case "float64":
			value, err = c.Command.Flags().GetFloat64(flagName)
		case "stringSlice":
			value, err = c.Command.Flags().GetStringSlice(flagName)
		default:
			value = flag.Value.String()
		}

		if err != nil {
			return fmt.Errorf("获取标志 %s 值失败: %v", flagName, err)
		}

		for _, validator := range validators {
			if err := validator.Validate(value); err != nil {
				return fmt.Errorf("参数 %s 验证失败: %v", flagName, err)
			}
		}
	}

	return nil
}

// AddParamValidator 为命令的特定标志添加参数校验器
func (c *Command) AddParamValidator(flagName string, validator ParamValidator) {
	if c.validators == nil {
		c.validators = make(map[string][]ParamValidator)
	}
	c.validators[flagName] = append(c.validators[flagName], validator)
}

// GetParamValidators 获取指定标志的所有校验器
func (c *Command) GetParamValidators(flagName string) []ParamValidator {
	if c.validators == nil {
		return nil
	}
	return c.validators[flagName]
}

// ClearParamValidators 清除指定标志的所有校验器
func (c *Command) ClearParamValidators(flagName string) {
	if c.validators != nil {
		delete(c.validators, flagName)
	}
}
