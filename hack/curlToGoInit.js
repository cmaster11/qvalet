const curlToGo = require('./curlToGo').curlToGo;

function main() {
  const args = process.argv.slice(2);
  const curlCommand = args[0];
  const goCode = curlToGo(curlCommand);

  const code = `
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func main() {
${goCode}
}
  `;

  process.stdout.write(code);
}

if (require.main === module) {
  main();
}