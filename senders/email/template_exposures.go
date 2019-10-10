package email

const exposuresTemplate = `
<!DOCTYPE html>
<html>

<head>
  <title></title>
  <meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta http-equiv="X-UA-Compatible" content="IE=edge" />
  <style type="text/css">
    body,
    table,
    td,
    a {
      -webkit-text-size-adjust: 100%;
      -ms-text-size-adjust: 100%;
    }
    a {
      font-family: 'Cambria';
    }
    p {
      color: #111111;
      font-family: 'Cambria';
    }

    table,
    td {
      font-size: 14px;
      mso-table-lspace: 0pt;
      mso-table-rspace: 0pt;
    }

    img {
      -ms-interpolation-mode: bicubic;
    }

    /* RESET STYLES */

    img {
      border: 0;
      height: auto;
      line-height: 100%;
      outline: none;
      text-decoration: none;
    }

    table {
      border-collapse: collapse !important;
    }

    body {
      height: 100% !important;
      margin: 0 !important;
      padding: 0 !important;
      width: 100% !important;
    }

    /* iOS BLUE LINKS */
    a[x-apple-data-detectors] {
      color: inherit !important;
      text-decoration: none !important;
      font-size: inherit !important;
      font-family: inherit !important;
      font-weight: inherit !important;
      line-height: inherit !important;
    }

    /* MOBILE STYLES */

    @media screen and (max-width:600px) {
      h1 {
        font-size: 32px !important;
        line-height: 32px !important;
      }
    }

    /* ANDROID CENTER FIX */

    div[style*="margin: 16px 0;"] {
      margin: 0 !important;
    }
  </style>
</head>
<body style="background-color: #f4f4f4; margin: 0 !important; padding: 0 !important;">
  <table border="0" cellpadding="0" cellspacing="0" width="100%">
    <!-- LOGO -->
    <tr>
      <td bgcolor="#FFA73B" align="center">
        <!--[if (gte mso 9)|(IE)]>
            <table align="center" border="0" cellspacing="0" cellpadding="0" width="600">
            <tr>
            <td align="center" valign="top" width="600">
            <![endif]-->
        <table border="0" cellpadding="0" cellspacing="0" width="100%" style="max-width: 600px;">
          <tr>
            <td align="center" valign="top" style="padding: 0px 10px 0px 10px;">
              <p style="color: white; font-size:20pt">GitLab Security Alert</p>
            </td>
          </tr>
        </table>
        <!--[if (gte mso 9)|(IE)]>
            </td>
            </tr>
            </table>
            <![endif]-->
      </td>
    </tr>
    <!-- COPY BLOCK -->
    <tr>
      <td bgcolor="#f4f4f4" align="center" style="padding: 0px 10px 0px 10px;">
        <!--[if (gte mso 9)|(IE)]>
            <table align="center" border="0" cellspacing="0" cellpadding="0" width="600">
            <tr>
            <td align="center" valign="top" width="600">
            <![endif]-->
        <table border="0" cellpadding="0" cellspacing="0" width="100%" style="max-width: 600px;">
          <!-- TEXT -->
          <tr>
            <td bgcolor="#ffffff" align="left" style="padding: 30px 30px 0px 30px; font-size: 16px;">
              <p>Обнаружена зависимость, имеющая известную уязвимость</p>
              <p>Если это ошибка, ответь на это письмо чтобы мы добавили это в исключения.</p>
              <p> Найдено {{ .ExposuresCount }} уязвимых зависимостей в {{ .FilesCount }} файлах.</p>
            </td>
          </tr>
      </td>
      {{ range .Repos }}
      <tr>
        <td bgcolor="#ffffff" align="left" style="padding: 30px 30px 10px 30px;">
          <a style="color: rgb(216, 119, 0); font-size: 22px; font-weight: normal" href="{{ .RepoURL }}">{{ .RepoURL }}</a>
          <hr align="center" size="1" color="#111111" />
        </td>
      </tr>
      <!-- LIST OF VULNERABILITIES -->
      {{ range .Items }}
      <tr>
        <td bgcolor="#ffffff" align="left" style="padding: 0px 30px 0px 30px; font-size: 14px;">
          <p><a style="color: rgb(216, 119, 0); font-size: 14px;" href="{{ .RepoURL }}/blob/{{ .CommitHash }}/{{ .FilePath }}">{{ .FilePath }}</a> : {{ .DependencyName }}, {{ .Version }}
          </p>
              {{ range .Vulnerabilities }}
                <a style="color: rgb(216, 119, 0); font-size: 14px;" href="{{ .Reference }}">{{ .Title }}</a>, CVSS score: {{ .CvssScore }}
                <div>
                    <details>
                        <summary>Description</summary>
                        <div>{{ .Description }}</div>
                    </details>
                </div>
              {{ end }}
          <p style="font-size: 12px; text-align: right;">Commit
            <i>{{ .CommitHash }}</i> by
            <a style="color:rgb(216, 119, 0);" href="mailto:{{ .CommitEmail }}">{{ .CommitAuthor }}</a> ({{ .TimeStamp.Format "15:04:05 02.01.2006" }})</p>
        </td>
      </tr>
      {{ end }}
      <tr>
        <td bgcolor="#ffffff" align="left" style="padding: 30px 30px 0px 30px;"></td>
      </tr>
      {{ end }}
      <tr>
        <td bgcolor="#ffffff" align="left" style="padding: 20px 30px 20px 30px; border-radius: 0px 0px 4px 4px; font-size: 18px;">
          <p style="margin: 0;">Отдел безопасности веб-сервисов</p>
        </td>
      </tr>
      </table>
      <!--[if (gte mso 9)|(IE)]>
            </td>
            </tr>
            </table>
            <![endif]-->
      </td>
    </tr>
    <!-- FOOTER -->
    <tr>
      <td bgcolor="#f4f4f4" align="center" style="padding: 40px 10px 40px 10px;">
        <p style="margin: 0;">Не хочу больше получать такие письма,
        <a href="#" target="_blank" style="color: #111111; font-weight: 700;">отписаться</a>.</p>
      </td>
    </tr>
  </table>
</body>
</html>
`
