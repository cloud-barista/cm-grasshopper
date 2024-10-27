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
        "license": {
            "name": "Apache 2.0",
            "url": "http://www.apache.org/licenses/LICENSE-2.0.html"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/readyz": {
            "get": {
                "description": "Check Grasshopper is ready",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "[Admin]\tSystem management"
                ],
                "summary": "Check Ready",
                "operationId": "health-check-readyz",
                "responses": {
                    "200": {
                        "description": "Successfully get ready state.",
                        "schema": {
                            "$ref": "#/definitions/github_com_cloud-barista_cm-grasshopper_pkg_api_rest_model.SimpleMsg"
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
        "/software/execution_list": {
            "post": {
                "description": "Get software migration execution list.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "[Software]"
                ],
                "summary": "Get Execution List",
                "operationId": "get-execution-list",
                "parameters": [
                    {
                        "description": "Software info list",
                        "name": "getExecutionListReq",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/github_com_cloud-barista_cm-grasshopper_pkg_api_rest_model.GetExecutionListReq"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Successfully get migration execution list.",
                        "schema": {
                            "$ref": "#/definitions/github_com_cloud-barista_cm-grasshopper_pkg_api_rest_model.GetExecutionListRes"
                        }
                    },
                    "400": {
                        "description": "Sent bad request.",
                        "schema": {
                            "$ref": "#/definitions/github_com_cloud-barista_cm-grasshopper_pkg_api_rest_common.ErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Failed to get migration execution list.",
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
                "operationId": "install-software",
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
        },
        "/software/register": {
            "post": {
                "description": "Register the software.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "[Software]"
                ],
                "summary": "Register Software",
                "operationId": "register-software",
                "parameters": [
                    {
                        "description": "Software info",
                        "name": "softwareRegisterReq",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/github_com_cloud-barista_cm-grasshopper_pkg_api_rest_model.SoftwareRegisterReq"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Successfully registered the software.",
                        "schema": {
                            "$ref": "#/definitions/github_com_cloud-barista_cm-grasshopper_pkg_api_rest_model.SoftwareRegisterReq"
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
        },
        "/software/{softwareId}": {
            "delete": {
                "description": "Delete the software.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "[Software]"
                ],
                "summary": "Delete Software",
                "operationId": "delete-software",
                "parameters": [
                    {
                        "type": "string",
                        "description": "ID of the software.",
                        "name": "softwareId",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Successfully update the software",
                        "schema": {
                            "$ref": "#/definitions/github_com_cloud-barista_cm-grasshopper_pkg_api_rest_model.SimpleMsg"
                        }
                    },
                    "400": {
                        "description": "Sent bad request.",
                        "schema": {
                            "$ref": "#/definitions/github_com_cloud-barista_cm-grasshopper_pkg_api_rest_common.ErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Failed to delete the software",
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
        "github_com_cloud-barista_cm-grasshopper_pkg_api_rest_model.Execution": {
            "type": "object",
            "properties": {
                "order": {
                    "type": "integer"
                },
                "software_id": {
                    "type": "string"
                },
                "software_install_type": {
                    "type": "string"
                },
                "software_name": {
                    "type": "string"
                },
                "software_version": {
                    "type": "string"
                }
            }
        },
        "github_com_cloud-barista_cm-grasshopper_pkg_api_rest_model.GetExecutionListReq": {
            "type": "object",
            "required": [
                "architecture",
                "os",
                "os_version",
                "software_info_list"
            ],
            "properties": {
                "architecture": {
                    "type": "string"
                },
                "os": {
                    "type": "string"
                },
                "os_version": {
                    "type": "string"
                },
                "software_info_list": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/github_com_cloud-barista_cm-grasshopper_pkg_api_rest_model.SoftwareInfo"
                    }
                }
            }
        },
        "github_com_cloud-barista_cm-grasshopper_pkg_api_rest_model.GetExecutionListRes": {
            "type": "object",
            "properties": {
                "errors": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "execution_list": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/github_com_cloud-barista_cm-grasshopper_pkg_api_rest_model.Execution"
                    }
                }
            }
        },
        "github_com_cloud-barista_cm-grasshopper_pkg_api_rest_model.SimpleMsg": {
            "type": "object",
            "properties": {
                "message": {
                    "type": "string"
                }
            }
        },
        "github_com_cloud-barista_cm-grasshopper_pkg_api_rest_model.SoftwareInfo": {
            "type": "object",
            "required": [
                "name",
                "version"
            ],
            "properties": {
                "name": {
                    "type": "string"
                },
                "version": {
                    "type": "string"
                }
            }
        },
        "github_com_cloud-barista_cm-grasshopper_pkg_api_rest_model.SoftwareInstallReq": {
            "type": "object",
            "required": [
                "software_ids",
                "target"
            ],
            "properties": {
                "software_ids": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "target": {
                    "$ref": "#/definitions/github_com_cloud-barista_cm-grasshopper_pkg_api_rest_model.Target"
                }
            }
        },
        "github_com_cloud-barista_cm-grasshopper_pkg_api_rest_model.SoftwareInstallRes": {
            "type": "object",
            "properties": {
                "execution_id": {
                    "type": "string"
                },
                "execution_list": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/github_com_cloud-barista_cm-grasshopper_pkg_api_rest_model.Execution"
                    }
                }
            }
        },
        "github_com_cloud-barista_cm-grasshopper_pkg_api_rest_model.SoftwareRegisterReq": {
            "type": "object",
            "required": [
                "architecture",
                "install_type",
                "match_names",
                "name",
                "needed_packages",
                "os",
                "os_version",
                "version"
            ],
            "properties": {
                "architecture": {
                    "type": "string"
                },
                "install_type": {
                    "type": "string"
                },
                "match_names": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "name": {
                    "type": "string"
                },
                "needed_packages": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "os": {
                    "type": "string"
                },
                "os_version": {
                    "type": "string"
                },
                "version": {
                    "type": "string"
                }
            }
        },
        "github_com_cloud-barista_cm-grasshopper_pkg_api_rest_model.Target": {
            "type": "object",
            "required": [
                "mcis_id",
                "namespace_id",
                "vm_id"
            ],
            "properties": {
                "mcis_id": {
                    "type": "string"
                },
                "namespace_id": {
                    "type": "string"
                },
                "vm_id": {
                    "type": "string"
                }
            }
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "latest",
	Host:             "",
	BasePath:         "/grasshopper",
	Schemes:          []string{},
	Title:            "CM-Grasshopper REST API",
	Description:      "Software migration management module",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
