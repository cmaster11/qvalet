const curlToGo = require('./curlToGo').curlToGo;

function main() {
  const entries = [
    `curl "http://localhost:7055/hello" -H 'Content-Type: application/yaml' -d $'- name: Ragnar\\n- name: Rollo'`
  ]

  entries.forEach((e) => {
    const goCode = curlToGo(e);
    process.stdout.write(goCode);
  });

}

main();