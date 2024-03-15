package model

import (
	"encoding/json"
	"github.com/tidwall/gjson"
)

const (
	AUTHORITY_INFO_ATTR        string = "34f9569d-eba1-423a-a0c2-995e9c15665d"
	URL_ATTR                   string = "8a68156a-d1f5-4322-b2a5-26e872a6fc0e"
	CREDENTIAL_TYPE_ATTR       string = "85197836-2ceb-4e77-b14e-53d2e9761cfc"
	GROUP_CREDENTIAL_TYPE_ATTR string = "335aede7-dd1f-4c87-9ff8-7dc93f18c5fe"
	JWT_TOKEN_ATTR             string = "924a5013-0aee-4c3f-ac59-420bf68b230c"
	ROLE_ID_ATTR               string = "97a46e73-bf7d-421d-ae5a-2d0f453eb300"
	ROLE_SECRET_ATTR           string = "60daa99e-5b08-4f36-8f51-d136ecba74e9"
	AUTHORITY_ATTR             string = "24531b64-efd2-4a16-8ba8-ffef90890356"
	RA_PROFILE_ENGINE_ATTR     string = "e7817459-41cf-40d4-ad3d-9808ef14cad7"
	RA_PROFILE_ROLE_ATTR       string = "389dfa3c-cf45-458e-bca4-507d11b2858c"
	RA_PROFILE_AUTHORITY_ATTR  string = "5af5693a-74bf-4ec4-b101-44ce35d8455b"
)

type AttributeName string

const (
	KUBERNETES_CRED string = "kubernetes"
	APPROLE_CRED    string = "role"
	JWTOIDC_CRED    string = "jwt"
)

type CredentialType string

func GetCredentialTypeByName(credentialType string) AttributeContent {
	for _, attribute := range GetCredentialTypes() {
		if attribute.GetData() == credentialType {
			return attribute
		}
	}
	return nil
}
func GetCredentialTypes() []AttributeContent {
	return []AttributeContent{

		StringAttributeContent{
			Reference: "Kubernetes",
			Data:      KUBERNETES_CRED,
		}, StringAttributeContent{
			Reference: "AppRole",
			Data:      APPROLE_CRED,
		}, StringAttributeContent{
			Reference: "JWT/OIDC",
			Data:      JWTOIDC_CRED,
		},
	}
}

func GetAttributeDefByUUID(uuid string) Attribute {
	for _, attr := range GetAttributeList() {
		if attr.GetUuid() == uuid {
			return attr
		}
	}
	return nil
}

const (
	AuthorityManagementAttributes string = "AuthorityManagementAttributes"
	DisoveryAttributes            string = "DiscoveryAttributes"
	RAProfilesAttributes          string = "RAProfilesAttributes"
)

func GetAttributeListBySet(attributeSet string) []Attribute {
	switch attributeSet {
	case AuthorityManagementAttributes:
		return getAuthorityManagementAttributes()
	case DisoveryAttributes:
		return getDiscoveryAttributes()
	case RAProfilesAttributes:
		return getRAProfilesAttributes()
	}

	return nil
}

func GetAttributeList() []Attribute {
	attributeList := append(getAuthorityManagementAttributes(), getDiscoveryAttributes()...)
	attributeList = append(attributeList, getRAProfilesAttributes()...)
	attributeList = append(attributeList, getAuthorityManagementAttributes()...)
	return attributeList
}

func GetAtributeByUUID(uuid string) AttributeDefinition {
	for _, attr := range GetAttributeList() {
		if attr.GetUuid() == uuid {
			return AttributeDefinition{
				Uuid:                 attr.GetUuid(),
				AttributeType:        attr.GetAttributeType(),
				AttributeContentType: attr.GetAttributeContentType(),
			}
		}
	}
	return AttributeDefinition{}
}

func unmarshalAttributeContent(content []byte, contentType AttributeContentType) AttributeContent {
	var result AttributeContent
	switch contentType {
	case STRING:
		stringContent := StringAttributeContent{}
		err := json.Unmarshal(content, &stringContent)
		result = stringContent
		if err != nil {
			panic(err)
		}
	case OBJECT:
		objectContent := ObjectAttributeContent{}
		err := json.Unmarshal(content, &objectContent)
		result = objectContent
		if err != nil {
			panic(err)
		}
	case BOOLEAN:
		booleanContent := BooleanAttributeContent{}
		err := json.Unmarshal(content, &booleanContent)
		result = booleanContent
		if err != nil {
			panic(err)
		}

	case SECRET:
		secretAttributeContent := SecretAttributeContent{}
		err := json.Unmarshal(content, &secretAttributeContent)
		result = secretAttributeContent
		if err != nil {
			panic(err)
		}
	}

	return result
}

func unmarshalAttribute(content []byte, attrDef AttributeDefinition) Attribute {
	var result Attribute
	switch attrDef.AttributeType {
	case DATA:

		data := DataAttribute{}
		contents := gjson.GetBytes(content, "content")
		for _, content := range contents.Array() {
			data.Content = append(data.Content, unmarshalAttributeContent([]byte(content.Raw), attrDef.AttributeContentType))
		}
		data.Uuid = gjson.GetBytes(content, "uuid").String()
		data.Name = gjson.GetBytes(content, "name").String()
		data.Description = gjson.GetBytes(content, "description").String()
		data.Type = attrDef.AttributeType
		data.ContentType = attrDef.AttributeContentType
		properties := gjson.GetBytes(content, "properties").Raw
		if properties != "" {
			err := json.Unmarshal([]byte(properties), &data.Properties)
			if err != nil {
				panic(err)
			}
		}
		constraints := gjson.GetBytes(content, "constrains").Raw
		if constraints != "" {
			err := json.Unmarshal([]byte(constraints), &data.Constraints)
			if err != nil {
				panic(err)
			}
		}
		callbacks := gjson.GetBytes(content, "attributeCallback").Raw
		if callbacks != "" {
			err := json.Unmarshal([]byte(callbacks), &data.AttributeCallback)
			if err != nil {
				panic(err)
			}
		}
		result = data
	}

	return result
}

func unmarshalAttributeValue(content []byte, attrDef AttributeDefinition) Attribute {
	var result Attribute
	switch attrDef.AttributeType {
	case DATA:
		data := GetAttributeDefByUUID(attrDef.Uuid).(DataAttribute)
		contents := gjson.GetBytes(content, "content")
		data.Content = []AttributeContent{}
		for _, content := range contents.Array() {
			data.Content = append(data.Content, unmarshalAttributeContent([]byte(content.Raw), attrDef.AttributeContentType))
		}
		result = data
	}

	return result
}

func UnmarshalAttributesValues(content []byte) []Attribute {
	attributes := gjson.GetBytes(content, "@values")
	var result []Attribute
	for _, attribute := range attributes.Array() {
		def := GetAtributeByUUID(gjson.Get(attribute.Raw, "uuid").String())
		attributeObject := unmarshalAttributeValue([]byte(attribute.Raw), def)
		result = append(result, attributeObject)
	}
	return result
}

func UnmarshalAttributes(content []byte) []Attribute {
	attributes := gjson.GetBytes(content, "@values")
	var result []Attribute
	for _, attribute := range attributes.Array() {
		definition := AttributeDefinition{}
		err := json.Unmarshal([]byte(attribute.Raw), &definition)
		if err != nil {
			return nil
		}
		if definition.AttributeType == "" || definition.AttributeContentType == "" {
			def := GetAtributeByUUID(definition.Uuid)
			definition.AttributeType = def.AttributeType
			definition.AttributeContentType = def.AttributeContentType
		}
		attributeObject := unmarshalAttribute([]byte(attribute.Raw), definition)
		result = append(result, attributeObject)
	}
	return result
}

func GetAttributeFromArrayByUUID(uuid string, attributes []Attribute) Attribute {
	for _, attr := range attributes {
		if attr.GetUuid() == uuid {
			return attr
		}
	}
	return nil
}

func getRAProfilesAttributes() []Attribute {
	return []Attribute{
		DataAttribute{
			Uuid:        RA_PROFILE_ENGINE_ATTR,
			Name:        "ra_profile_engine",
			Description: "Selection of RA Profile engine",
			Type:        DATA,
			Content:     nil,
			ContentType: OBJECT,
			Properties: &DataAttributeProperties{
				Label:       "RA Profile PKI engine selection",
				Visible:     true,
				Group:       "",
				Required:    true,
				ReadOnly:    false,
				List:        false,
				MultiSelect: false,
			},
		},
		DataAttribute{
			Uuid:        RA_PROFILE_AUTHORITY_ATTR,
			Name:        "ra_profile_authority",
			Description: "",
			Type:        DATA,
			Content:     nil,
			ContentType: OBJECT,
			Properties: &DataAttributeProperties{
				Label:       "Authority used for PKI engine query",
				Visible:     false,
				Group:       "",
				Required:    true,
				ReadOnly:    false,
				List:        false,
				MultiSelect: false,
			},
		},
		DataAttribute{
			Uuid:        RA_PROFILE_ROLE_ATTR,
			Name:        "ra_profile_role",
			Description: "Select RA Profile Role",
			Type:        DATA,
			Content:     nil,
			ContentType: STRING,
			Properties: &DataAttributeProperties{
				Label:       "RA profile role",
				Visible:     true,
				Group:       "",
				Required:    true,
				ReadOnly:    false,
				List:        true,
				MultiSelect: false,
			},
			AttributeCallback: &AttributeCallback{
				CallbackContext: "/v1/authorityProvider/authorities/{uuid}/raProfileRole/{engineName}/callback",
				CallbackMethod:  "GET",
				Mappings: []AttributeCallbackMapping{
					{
						From:                 "ra_profile_engine.data.engineName",
						AttributeType:        DATA,
						AttributeContentType: STRING,
						To:                   "engineName",
						Targets: []AttributeValueTarget{
							PATH_VARIABLE,
						},
					},
					{
						From:                 "ra_profile_authority.uuid",
						AttributeType:        DATA,
						AttributeContentType: STRING,
						To:                   "uuid",
						Targets: []AttributeValueTarget{
							PATH_VARIABLE,
						},
					},
				},
			},
		},
	}
}

func getDiscoveryAttributes() []Attribute {
	return []Attribute{
		DataAttribute{
			Uuid:        AUTHORITY_ATTR,
			Name:        "authority_to_discover",
			Description: "Authority definition for discovery",
			Type:        DATA,
			Content:     nil,
			ContentType: OBJECT,
			Properties: &DataAttributeProperties{
				Label:       "Authority to discover",
				Visible:     true,
				Group:       "",
				Required:    true,
				ReadOnly:    false,
				List:        false,
				MultiSelect: false,
			},
		},
	}

}

func getAuthorityManagementAttributes() []Attribute {
	return []Attribute{
		InfoAttribute{
			Uuid:        AUTHORITY_INFO_ATTR,
			Name:        "authority_info",
			Description: "Authority definition for discovery",
			Type:        INFO,
			ContentType: TEXT,
			Content: []AttributeContent{
				TextAttributeContent{
					Data: `### HashiCorp Vault instance configuration.

Provide URL of your Vault and select one of the available authentication methods:
-  **AppRole** - Use AppRole authentication method with Role ID and Secret ID
-  **Kubernetes** - Use Kubernetes authentication method with Service Account Token (automatically taken from the environment)
-  **JWT/OIDC** - Use JWT/OIDC authentication method with provided JWT token (automatically taken from the environment)
`,
				},
			},
			Properties: InfoAttributeProperties{
				Label:   "Authority Info",
				Visible: true,
				Group:   "",
			},
		},
		DataAttribute{
			Uuid:        URL_ATTR,
			Name:        "authority_url",
			Description: "Vault URL for authority",
			Type:        DATA,
			Content:     []AttributeContent{},
			ContentType: STRING,
			Properties: &DataAttributeProperties{
				Label:       "Vault URL",
				Visible:     true,
				Group:       "",
				Required:    true,
				ReadOnly:    false,
				List:        false,
				MultiSelect: false,
			},
			Constraints: []AttributeConstraint{
				RegexpAttributeConstraint{
					Description:  "URL for the HashiCorp Vault",
					ErrorMessage: "URL must be a valid URL",
					Type:         REG_EXP,
					Data:         "^(http|https)://[a-zA-Z0-9.-]+(:[0-9]+)?",
				},
			},
		},
		DataAttribute{
			Uuid:        CREDENTIAL_TYPE_ATTR,
			Name:        "credentials_type",
			Description: "Credentials type for authority connection",
			Type:        DATA,
			Content:     nil,
			ContentType: STRING,
			Properties: &DataAttributeProperties{
				Label:       "Credential type",
				Visible:     true,
				Group:       "",
				Required:    true,
				ReadOnly:    false,
				List:        true,
				MultiSelect: false,
			},
		},
		GroupAttribute{
			Uuid:        GROUP_CREDENTIAL_TYPE_ATTR,
			Name:        "credential_group",
			Description: "Authority definition for discovery",
			Type:        GROUP,
			AttributeCallback: &AttributeCallback{
				CallbackContext: "v1/authorityProvider/credentialType/{credentialsType}/callback",
				CallbackMethod:  "GET",
				Mappings: []AttributeCallbackMapping{
					{
						From:                 "credentials_type.data",
						AttributeType:        DATA,
						AttributeContentType: STRING,
						To:                   "credentialsType",
						Targets: []AttributeValueTarget{
							PATH_VARIABLE,
						},
					},
				},
			},
		},
		DataAttribute{
			Uuid:        ROLE_ID_ATTR,
			Name:        "role_id",
			Description: "RoleId for connection to vault",
			Type:        DATA,
			Content:     nil,
			ContentType: SECRET,
			Properties: &DataAttributeProperties{
				Label:       "Role ID",
				Visible:     true,
				Group:       "",
				Required:    true,
				ReadOnly:    false,
				List:        true,
				MultiSelect: false,
			},
		},
		DataAttribute{
			Uuid:        ROLE_SECRET_ATTR,
			Name:        "role_secret",
			Description: "RoleSecret for connection to vault",
			Type:        DATA,
			Content:     nil,
			ContentType: SECRET,
			Properties: &DataAttributeProperties{
				Label:       "Role Secret",
				Visible:     true,
				Group:       "",
				Required:    true,
				ReadOnly:    false,
				List:        true,
				MultiSelect: false,
			},
		},
		DataAttribute{
			Uuid:        JWT_TOKEN_ATTR,
			Name:        "jwt_token",
			Description: "JWT Token for connection to vault",
			Type:        DATA,
			Content:     nil,
			ContentType: SECRET,
			Properties: &DataAttributeProperties{
				Label:       "JWT Token",
				Visible:     true,
				Group:       "",
				Required:    true,
				ReadOnly:    false,
				List:        true,
				MultiSelect: false,
			},
		},
	}
}
