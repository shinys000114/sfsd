package handler

const defaultDirTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Index of {{.Path}}</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; margin: 0; padding: 0; background: #fff; }
        
        .markdown-wrapper {
            max-width: 980px;
			margin: 10px;
            margin-bottom: 40px;
            padding: 20px;
            border: 1px solid #ccc;
            border-radius: 6px;
        }

		.dir-listing {
			padding: 20px;
		}

        .dir-listing h1 { font-size: 1.5rem; margin-bottom: 20px; color: #333; }
        .dir-listing table { width: 100%; max-width: 1000px; border-collapse: collapse; text-align: left; font-size: 14px; }
        .dir-listing th, .dir-listing td { padding: 8px 10px; border-bottom: 1px solid #eaeaea; }
        .dir-listing th { font-weight: 600; color: #444; background-color: #f6f8fa; }
        .dir-listing tr:hover td { background-color: #f8f8f8; }
        
        .dir-listing a { text-decoration: none; color: #0366d6; }
        .dir-listing a:hover { text-decoration: underline; }
        
        .dir-listing .icon { width: 20px; text-align: center; }
        .dir-listing .size { text-align: right; color: #6a737d; width: 100px; font-family: monospace; }
        .dir-listing .date { color: #6a737d; width: 160px; text-align: right; }

        @media (max-width: 600px) {
            .dir-listing .date, .dir-listing .size { display: none; }
        }
    </style>
</head>
<body>
    {{if .MdHtml}}
    <div class="markdown-wrapper">
        <article class="markdown-body">
            {{.MdHtml}}
        </article>
    </div>
    {{end}}

    <div class="dir-listing">
        <h1>Index of {{.Path}}</h1>
        <table>
            <thead>
                <tr>
                    <th class="icon"></th>
                    <th>Name</th>
                    <th class="size">Size</th>
                    <th class="date">Last Modified</th>
                </tr>
            </thead>
            <tbody>
                {{if ne .Path "/"}}
                <tr>
                    <td class="icon">{{.DirectoryIcon}}</td>
                    <td><a href="../">..</a></td>
                    <td class="size">-</td>
                    <td class="date">-</td>
                </tr>
                {{end}}
                {{range .Files}}
                <tr>
                    <td class="icon">{{.Icon}}</td>
                    <td><a href="{{.URL}}">{{.Name}}</a></td>
                    <td class="size">{{.Size}}</td>
                    <td class="date">{{.ModTime}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </div>
</body>
</html>`
