# GH_to_doc

## Examples

### BeerCSS


```shell
GITHUB_TOKEN=${GITHUB_API_TOKEN} go run . \
  --repo-url  https://github.com/beercss/beercss/tree/main/docs \
  --rules     beercss.rules.yaml \
  --out       beercss-docs-clean.md
```

#### JSpreadsheet
```shell
GITHUB_TOKEN=${GITHUB_API_TOKEN} go run . \
  --repo-url  https://github.com/jspreadsheet/ce/tree/master/docs \
  --rules     jspreadsheet.rules.yaml \
  --out       jspreadsheet-docs-clean.md
```

## License

This project is licensed under the [MIT License](./LICENSE).