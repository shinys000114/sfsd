package handler

const defaultDirTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Index of {{.Path}}</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; background: #fdfdfd; color: #333; margin: 0; padding: 20px; }
        h1 { font-size: 1.5rem; margin-bottom: 20px; word-wrap: break-word; }
        table { width: 100%; max-width: 1000px; border-collapse: collapse; text-align: left; }
        th, td { padding: 10px 0; border-bottom: 1px solid #ddd; }
        th { font-weight: 500; color: #555; }
        td a { text-decoration: none; color: #0066cc; font-weight: 500;}
        td a:hover { text-decoration: underline; }
        .size { text-align: right; color: #666; width: 120px; }
        .date { color: #666; width: 200px; }
        .icon { width: 24px; text-align: center; }
        @media (max-width: 600px) {
            .date, .size { display: none; }
        }
    </style>
</head>
<body>
    <h1>Index of {{.Path}}</h1>
    <table>
        <thead>
            <tr>
                <th class="icon"></th>
                <th>Name</th>
                <th class="date">Last Modified</th>
                <th class="size">Size</th>
            </tr>
        </thead>
        <tbody>
            {{if ne .Path "/"}}
            <tr>
                <td class="icon">📁</td>
                <td><a href="../">..</a></td>
                <td class="date">-</td>
                <td class="size">-</td>
            </tr>
            {{end}}
            {{range .Files}}
            <tr>
                <td class="icon">{{if .IsDir}}📁{{else}}📄{{end}}</td>
                <td><a href="{{.URL}}">{{.Name}}</a></td>
                <td class="date">{{.ModTime}}</td>
                <td class="size">{{.Size}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
</body>
</html>`
