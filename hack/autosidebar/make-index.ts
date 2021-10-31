// Create reference instance
import * as path from 'path';
import * as fs from "fs";

const marked = require('marked');

// Set options
// `highlight` example uses https://highlightjs.org
marked.setOptions({
    renderer: new marked.Renderer(),
    highlight: function (code: string, lang: string) {
        const hljs = require('highlight.js');
        const language = hljs.getLanguage(lang) ? lang : 'plaintext';
        return hljs.highlight(code, {language}).value;
    },
    langPrefix: 'hljs language-', // highlight.js css expects a top-level 'hljs' class.
    pedantic: false,
    gfm: true,
    breaks: false,
    sanitize: false,
    smartLists: true,
    smartypants: false,
    xhtml: false
});

const tpl = fs.readFileSync(path.join(__dirname, '..', '..', 'index.tpl.html'), {encoding: 'utf-8'});
const content = fs.readFileSync(path.join(__dirname, '..', '..', 'README.md'), {encoding: 'utf-8'});
const html = marked(content);
const out = tpl.replace('__REPLACE_MD__', html);

fs.writeFileSync(path.join(__dirname, '..', '..', 'index.html'), out, {encoding: 'utf-8'});