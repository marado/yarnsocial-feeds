package main

import (
	"bytes"
	text_template "text/template"
)

// RenderPlainText ...
func RenderPlainText(tpl string, ctx interface{}) (string, error) {
	t := text_template.Must(text_template.New("tpl").Parse(tpl))
	buf := bytes.NewBuffer([]byte{})
	err := t.Execute(buf, ctx)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

const preambleTemplate = `# Twtxt is an open, distributed microblogging platform that
# uses human-readable text files, common transport protocols,
# and free software.
#
# Learn more about twtxt at  https://github.com/buckket/twtxt
#
# This is an automated Yarn.social feed running feeds v{{ .SoftwareVersion }}
# Learn more about Yarn.social at https://yarn.social
#
# nick        = {{ .Name }}
# url         = {{ .URL }}
# type        = {{ .Type }}
{{ with .Source -}}
# source      = {{ . }}
{{ end -}}
# avatar      = {{ .Avatar }}
# description = {{ .Description }}
# updated_at  = {{ .LastModified }}
#
`

const indexTemplate = `
<!DOCTYPE html>
<html lang="en">
  <head>
    <link rel="stylesheet" href="https://unpkg.com/@picocss/pico@latest/css/pico.min.css">
    <meta name="viewport" content="width=device-width, initial-scale=1" />
	<meta name="yarn-uri" content="/%s/twtxt.txt" />
    <title>twtxtfeeds :: {{ .Title }}</title>
  </head>
<body>
  <nav class="container-fluid">
    <ul>
      <li><strong><a href="/">Twtxt Feeds</a></strong></li>
      <li><a href="/feeds">Feeds</a></li>
    </ul>
  </nav>
  <main class="container">
    <article class="grid">
      <div>
        <div class="container-fluid">
          <form action="/" method="POST">
			<input type="uri" id="uri" name="uri" placeholder="Enter any Website's URL, an RSS/Atom feed URI, or mastodon://handle" required>
            <div><button type="submit">Go!</button>
          </form>
        </div>
        <p>
          feeds is a command-line tool and web app that processes RSS and Atom feeds
          into <a href="https://twtxt.readthedocs.io/en/stable/index.html">twtxt</a>
          feeds for consumption by <i>twtxt</i> clients such as <a href="https://twtxt.net">twtxt.net</a>
          and <a href="https://yarn.social">Yarn.social</a> pods.
        </p>
        <p>
          You may freely create new feeds here by simply dropping a website's URL or any valid RSS/Atom URI.
        </p>
        <p>
          You are also welcome to subscribe to any of the <a href="/feeds">feeds</a>
          with your favorite <i>twtxt</i> client (<i>I like using
          <a href="https://github.com/quite/twet">twet</a></i>).
          Be sure to check out <a href="https://twtxt.net">twtxt.net</a>
          and <a href="https://yarn.social">Yarn.social</a> pods.
        </p>
      </div>
    </article>
  </main>
  <footer class="container-fluid">
    <hr>
    <p>
      <small>
        Licensed under the <a href="https://git.mills.io/yarnsocial/feeds/blob/master/LICENSE" class="secondary">MIT License</a><br>
      </small>
    </p>
  </footer>
</body>
</html>
`

const feedsTemplate = `
<!DOCTYPE html>
<html lang="en">
  <head>
    <link rel="stylesheet" href="https://unpkg.com/@picocss/pico@latest/css/pico.min.css">
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>twtxtfeeds :: {{ .Title }}</title>
  </head>
<body>
  <nav class="container-fluid">
    <ul>
      <li><strong><a href="/">feeds</a></strong></li>
      <li><a href="/feeds">Feeds</a></li>
    </ul>
  </nav>
  <main class="container">
    <article class="grid">
      <div>
        <hgroup>
          <h2>Feeds</h2>
          <footer>Available twtxt feeds</footer>
        </hgroup>
        {{ if .Feeds }}
          <ul>
            {{ range .Feeds }}
              <li><a href="{{ .URI }}">{{ .Name }}</a>&nbsp;<small>({{ .LastModified }})</small>: {{ .Description }}</li>
            {{ end }}
          </ul>
        {{ else }}
          <small>There are no feeds available yet. Come back later!</small>
        {{ end }}
      </div>
    </article>
  </main>
  <footer class="container-fluid">
    <hr>
    <p>
      <small>
        Licensed under the <a href="https://git.mills.io/yarnsocial/feeds/blob/master/LICENSE" class="secondary">MIT License</a><br>
      </small>
    </p>
  </footer>
</body>
</html>
`

const messageTemplate = `
<!DOCTYPE html>
<html lang="en">
  <head>
    <link rel="stylesheet" href="https://unpkg.com/@picocss/pico@latest/css/pico.min.css">
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>{{ .Title }}</title>
  </head>
<body>
  <nav class="container-fluid">
    <ul>
      <li><strong><a href="/">feeds</a></strong></li>
      <li><a href="/feeds">Feeds</a></li>
    </ul>
    <ul>
      <li>
        <a href="https://git.mills.io/yarnsocial/feeds" class="contrast" aria-label="Pico GitHub repository">
          <svg aria-hidden="true" focusable="false" role="img" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 496 512" height="1rem">
            <path fill="currentColor" d="M165.9 397.4c0 2-2.3 3.6-5.2 3.6-3.3.3-5.6-1.3-5.6-3.6 0-2 2.3-3.6 5.2-3.6 3-.3 5.6 1.3 5.6 3.6zm-31.1-4.5c-.7 2 1.3 4.3 4.3 4.9 2.6 1 5.6 0 6.2-2s-1.3-4.3-4.3-5.2c-2.6-.7-5.5.3-6.2 2.3zm44.2-1.7c-2.9.7-4.9 2.6-4.6 4.9.3 2 2.9 3.3 5.9 2.6 2.9-.7 4.9-2.6 4.6-4.6-.3-1.9-3-3.2-5.9-2.9zM244.8 8C106.1 8 0 113.3 0 252c0 110.9 69.8 205.8 169.5 239.2 12.8 2.3 17.3-5.6 17.3-12.1 0-6.2-.3-40.4-.3-61.4 0 0-70 15-84.7-29.8 0 0-11.4-29.1-27.8-36.6 0 0-22.9-15.7 1.6-15.4 0 0 24.9 2 38.6 25.8 21.9 38.6 58.6 27.5 72.9 20.9 2.3-16 8.8-27.1 16-33.7-55.9-6.2-112.3-14.3-112.3-110.5 0-27.5 7.6-41.3 23.6-58.9-2.6-6.5-11.1-33.3 2.6-67.9 20.9-6.5 69 27 69 27 20-5.6 41.5-8.5 62.8-8.5s42.8 2.9 62.8 8.5c0 0 48.1-33.6 69-27 13.7 34.7 5.2 61.4 2.6 67.9 16 17.7 25.8 31.5 25.8 58.9 0 96.5-58.9 104.2-114.8 110.5 9.2 7.9 17 22.9 17 46.4 0 33.7-.3 75.4-.3 83.6 0 6.5 4.6 14.4 17.3 12.1C428.2 457.8 496 362.9 496 252 496 113.3 383.5 8 244.8 8zM97.2 352.9c-1.3 1-1 3.3.7 5.2 1.6 1.6 3.9 2.3 5.2 1 1.3-1 1-3.3-.7-5.2-1.6-1.6-3.9-2.3-5.2-1zm-10.8-8.1c-.7 1.3.3 2.9 2.3 3.9 1.6 1 3.6.7 4.3-.7.7-1.3-.3-2.9-2.3-3.9-2-.6-3.6-.3-4.3.7zm32.4 35.6c-1.6 1.3-1 4.3 1.3 6.2 2.3 2.3 5.2 2.6 6.5 1 1.3-1.3.7-4.3-1.3-6.2-2.2-2.3-5.2-2.6-6.5-1zm-11.4-14.7c-1.6 1-1.6 3.6 0 5.9 1.6 2.3 4.3 3.3 5.6 2.3 1.6-1.3 1.6-3.9 0-6.2-1.4-2.3-4-3.3-5.6-2z"></path>
          </svg>
        </a>
      </li>
    </ul>
  </nav>
  <main class="container">
    <article class="grid">
      <div>
		<hgroup>
		  <h1>{{ .Title }}</h1>
          <p>{{ .Message }}</p>
	    </hgroup>
      </div>
    </article>
  </main>
  <footer class="container-fluid">
    <hr>
    <p>
      <small>
        Licensed under the <a href="https://git.mills.io/yarnsocial/feeds/blob/master/LICENSE" class="secondary">MIT License</a><br>
      </small>
    </p>
  </footer>
</body>
</html>
`
