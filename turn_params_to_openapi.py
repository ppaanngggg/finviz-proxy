"""
turn json from `/params` into openapi schema for `/table_v2`

usage:
python turn_params_to_openapi.py params.json > openapi.yaml

requirements:
- python3
- pip install pyyaml
"""

import json
import yaml
from pathlib import Path
from argparse import ArgumentParser


def main(file: Path):
    data = json.loads(file.read_text())
    openapi_spec = {
        "openapi": "3.0.0",
        "info": {
            "title": "Finviz Screener API",
            "version": "1.0.0",
            "description": "API for retrieving table data from Finviz Screener",
        },
        "servers": [
            {"url": "http://localhost:8000", "description": "local"},
        ],
        "paths": {
            "/table_v2": {
                "post": {
                    "summary": "Retrieve table data",
                    "requestBody": {
                        "required": True,
                        "content": {
                            "application/json": {
                                "schema": {
                                    "type": "object",
                                    "properties": {
                                        "order": {
                                            "type": "string",
                                            "enum": [
                                                sorter["value"]
                                                for sorter in data["sorters"]
                                            ],
                                            "description": "Sorting field",
                                        },
                                        "desc": {
                                            "type": "boolean",
                                            "description": "Sort in descending order if true",
                                        },
                                        "signal": {
                                            "type": "string",
                                            "enum": [
                                                signal["value"]
                                                for signal in data["signals"]
                                            ],
                                            "description": "Signal to filter by",
                                        },
                                        "filters": {"type": "object", "properties": {}},
                                    },
                                }
                            }
                        },
                    },
                    "responses": {
                        "200": {
                            "description": "Successful response",
                            "content": {
                                "application/json": {
                                    "schema": {
                                        "type": "object",
                                        "properties": {
                                            "headers": {
                                                "type": "array",
                                                "items": {"type": "string"},
                                                "description": "Column headers for the table",
                                            },
                                            "rows": {
                                                "type": "array",
                                                "items": {
                                                    "type": "array",
                                                    "items": {"type": "string"},
                                                },
                                                "description": "Table data rows",
                                            },
                                        },
                                    }
                                }
                            },
                        }
                    },
                }
            }
        },
    }

    # add filters into schema
    for filter in data["filters"]:
        filter_id = filter["id"]
        filter_options = [option["value"] for option in filter["options"]]
        openapi_spec["paths"]["/table_v2"]["post"]["requestBody"]["content"][
            "application/json"
        ]["schema"]["properties"]["filters"]["properties"][filter_id] = {
            "type": "string",
            "enum": filter_options,
            "description": filter["description"],
        }

    # write into yaml file
    yaml_spec = yaml.dump(openapi_spec, sort_keys=False)
    with open("openapi.yaml", "w") as f:
        f.write(yaml_spec)

    print("OpenAPI YAML specification has been generated and saved as 'openapi.yaml'.")


if __name__ == "__main__":
    parser = ArgumentParser()
    parser.add_argument("file", type=Path)
    args = parser.parse_args()
    main(args.file)
