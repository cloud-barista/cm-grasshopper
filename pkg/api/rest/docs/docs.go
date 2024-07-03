// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/grasshopper/readyz": {
            "get": {
                "description": "Check Grasshopper is ready",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "[Admin] System management"
                ],
                "summary": "Check Ready",
                "responses": {
                    "200": {
                        "description": "Successfully get ready state.",
                        "schema": {
                            "$ref": "#/definitions/pkg_api_rest_controller.SimpleMsg"
                        }
                    },
                    "500": {
                        "description": "Failed to check ready state.",
                        "schema": {
                            "$ref": "#/definitions/github_com_cloud-barista_cm-grasshopper_pkg_api_rest_common.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/software/install": {
            "post": {
                "description": "Install pieces of software to target.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "[Software]"
                ],
                "summary": "Install Software",
                "parameters": [
                    {
                        "description": "Software install request.",
                        "name": "softwareInstallReq",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/github_com_cloud-barista_cm-grasshopper_pkg_api_rest_model.SoftwareInstallReq"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Successfully sent SSH command.",
                        "schema": {
                            "$ref": "#/definitions/github_com_cloud-barista_cm-grasshopper_pkg_api_rest_model.SoftwareInstallRes"
                        }
                    },
                    "400": {
                        "description": "Sent bad request.",
                        "schema": {
                            "$ref": "#/definitions/github_com_cloud-barista_cm-grasshopper_pkg_api_rest_common.ErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Failed to sent SSH command.",
                        "schema": {
                            "$ref": "#/definitions/github_com_cloud-barista_cm-grasshopper_pkg_api_rest_common.ErrorResponse"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "github_com_cloud-barista_cm-grasshopper_pkg_api_rest_common.ErrorResponse": {
            "type": "object",
            "properties": {
                "error": {
                    "type": "string"
                }
            }
        },
        "github_com_cloud-barista_cm-grasshopper_pkg_api_rest_model.SoftwareInstallReq": {
            "type": "object",
            "required": [
                "connection_id",
                "package_names",
                "package_type"
            ],
            "properties": {
                "connection_id": {
                    "type": "string"
                },
                "package_names": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "package_type": {
                    "type": "string"
                }
            }
        },
        "github_com_cloud-barista_cm-grasshopper_pkg_api_rest_model.SoftwareInstallRes": {
            "type": "object",
            "properties": {
                "output": {
                    "type": "string"
                }
            }
        },
        "pkg_api_rest_controller.SimpleMsg": {
            "type": "object",
            "properties": {
                "message": {
                    "type": "string"
                }
            }
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "",
	Host:             "",
	BasePath:         "",
	Schemes:          []string{},
	Title:            "",
	Description:      "",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
