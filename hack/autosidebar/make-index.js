"use strict";
exports.__esModule = true;
// Create reference instance
var path = require("path");
var fs = require("fs");
var marked = require('marked');
// Set options
// `highlight` example uses https://highlightjs.org
marked.setOptions({
    renderer: new marked.Renderer(),
    highlight: function (code, lang) {
        var hljs = require('highlight.js');
        var language = hljs.getLanguage(lang) ? lang : 'plaintext';
        return hljs.highlight(code, { language: language }).value;
    },
    langPrefix: 'hljs language-',
    pedantic: false,
    gfm: true,
    breaks: false,
    sanitize: false,
    smartLists: true,
    smartypants: false,
    xhtml: false
});
var tpl = fs.readFileSync(path.join(__dirname, '..', '..', 'index.tpl.html'), { encoding: 'utf-8' });
var content = fs.readFileSync(path.join(__dirname, '..', '..', 'README.md'), { encoding: 'utf-8' });
var html = marked(content);
var out = tpl.replace('__REPLACE_MD__', html);
fs.writeFileSync(path.join(__dirname, '..', '..', 'index.html'), out, { encoding: 'utf-8' });
