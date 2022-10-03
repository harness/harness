# A demo of `react-markdown`

`react-markdown` is a markdown component for React.

👉 Changes are re-rendered as you type.

👈 Try writing some markdown on the left.

## Overview

- Follows [CommonMark](https://commonmark.org)
- Optionally follows [GitHub Flavored Markdown](https://github.github.com/gfm/)
- Renders actual React elements instead of using `dangerouslySetInnerHTML`
- Lets you define your own components (to render `MyHeading` instead of `h1`)
- Has a lot of plugins

## Table of contents

Here is an example of a plugin in action
([`remark-toc`](https://github.com/remarkjs/remark-toc)).
This section is replaced by an actual table of contents.

## Syntax highlighting

Here is an example of a plugin to highlight code:
[`rehype-highlight`](https://github.com/rehypejs/rehype-highlight).

```js
import React from 'react'
import ReactDOM from 'react-dom'
import ReactMarkdown from 'react-markdown'
import rehypeHighlight from 'rehype-highlight'

ReactDOM.render(
  <ReactMarkdown rehypePlugins={[rehypeHighlight]}>{'# Your markdown here'}</ReactMarkdown>,
  document.querySelector('#content')
)
```

Pretty neat, eh?

## GitHub flavored markdown (GFM)

For GFM, you can _also_ use a plugin:
[`remark-gfm`](https://github.com/remarkjs/react-markdown#use).
It adds support for GitHub-specific extensions to the language:
tables, strikethrough, tasklists, and literal URLs.

These features **do not work by default**.
👆 Use the toggle above to add the plugin.

|    Feature | Support              |
| ---------: | :------------------- |
| CommonMark | 100%                 |
|        GFM | 100% w/ `remark-gfm` |

~~strikethrough~~

- [ ] task list
- [x] checked item

https://example.com

## HTML in markdown

⚠️ HTML in markdown is quite unsafe, but if you want to support it, you can
use [`rehype-raw`](https://github.com/rehypejs/rehype-raw).
You should probably combine it with
[`rehype-sanitize`](https://github.com/rehypejs/rehype-sanitize).

<blockquote>
  👆 Use the toggle above to add the plugin.
</blockquote>

## Components

You can pass components to change things:

```js
import React from 'react'
import ReactDOM from 'react-dom'
import ReactMarkdown from 'react-markdown'
import MyFancyRule from './components/my-fancy-rule.js'

ReactDOM.render(
  <ReactMarkdown
    components={{
      // Use h2s instead of h1s
      h1: 'h2',
      // Use a component instead of hrs
      hr: ({ node, ...props }) => <MyFancyRule {...props} />
    }}>
    # Your markdown here
  </ReactMarkdown>,
  document.querySelector('#content')
)
```

### JSX

```jsx
import root from 'react-shadow'
import styles from './styles.css'

export default function Quote() {
  return (
    <root.div className="quote">
      <q>There is strong shadow where there is much light.</q>
      <span className="author">― Johann Wolfgang von Goethe.</span>
      <style type="text/css">{styles}</style>
    </root.div>
  )
}
```

### Java

```java
public static void main() {
  System.out.println('Hello World');
}
```

### CSS

```css
.main {
  background: white;
}
```

## More info?

Much more info is available in the
[readme on GitHub](https://github.com/remarkjs/react-markdown)!

---

A component by [Espen Hovlandsdal](https://espen.codes/)

![ReactShadow](https://github.com/Wildhoney/ReactShadow/raw/master/media/logo.png)

> Utilise Shadow DOM in React with all the benefits of style encapsulation.

![Travis](http://img.shields.io/travis/Wildhoney/ReactShadow.svg?style=flat-square)
&nbsp;
![Coveralls](https://img.shields.io/coveralls/Wildhoney/ReactShadow.svg?style=flat-square)
&nbsp;
![npm](http://img.shields.io/npm/v/react-shadow.svg?style=flat-square)
&nbsp;
![License MIT](https://img.shields.io/badge/license-MIT-lightgrey.svg?style=flat-square)

- **npm**: `npm i react-shadow`
- **yarn**: `yarn add react-shadow`
- **Heroku**: [https://react-shadow.herokuapp.com/](https://react-shadow.herokuapp.com) ([alternative](https://react-shadow-2.herokuapp.com))

<img src="https://github.com/Wildhoney/ReactShadow/raw/master/media/screenshot.png" width1="50%">

---

## Getting Started

Creating the [shadow root](https://www.w3.org/TR/shadow-dom/) is as simple as using the default export to construct a shadow root using the node name provided &ndash; for example `root.div` would create a `div` as the host element, and a shadow root as its immediate descendant &mdash; all of the child elements would then be descendants of the shadow boundary.

```jsx
import root from 'react-shadow'
import styles from './styles.css'

export default function Quote() {
  return (
    <root.div className="quote">
      <q>There is strong shadow where there is much light.</q>
      <span className="author">― Johann Wolfgang von Goethe.</span>
      <style type="text/css">{styles}</style>
    </root.div>
  )
}
```

```java
public static void main() {
  System.out.println('Hello World');
}
```

[![Edit react-shadow](https://codesandbox.io/static/img/play-codesandbox.svg)](https://codesandbox.io/s/react-shadow-by6bo?fontsize=14)

Applying styles requires either applying the styles directly to the component as a string, or importing the CSS documents as a string as part of your build process. You can then append the `style` component directly to your shadow boundary via your component's tree. In [the example](https://github.com/Wildhoney/ReactShadow/tree/master/example) we use the following Webpack configuration to import CSS documents as strings.

```javascript
{
    test: /\.css$/,
    loader: ['to-string-loader', 'css-loader']
}
```

Alternatively you can use [`styled-components`](https://www.styled-components.com/) normally, as each time a shadow boundary is created, a new `StyleSheetManager` context is also created which will encapsulate all related styles in their corresponding shadow root &mdash; to use this `import react-shadow/styled-components` instead of `import react-shadow`, likewise if you'd like to use [`emotion`](https://emotion.sh/docs/styled) you can `import react-shadow/emotion`.

```javascript
import root from 'react-shadow/styled-components'
import root from 'react-shadow/emotion'

// ...
;<root.section />
```

You may pass any props you like to the `root.*` component which will be applied directly to the host element, including event handlers and class names. There are also a handful of options that are used for the `attachShadow` invocation.

```javascript
ShadowRoot.propTypes = {
  mode: PropTypes.oneOf(['open', 'closed']),
  delegatesFocus: PropTypes.bool,
  styleSheets: PropTypes.arrayOf(PropTypes.instanceOf(globalThis.CSSStyleSheet)),
  children: PropTypes.node
}

ShadowRoot.defaultProps = {
  mode: 'open',
  delegatesFocus: false,
  styleSheets: [],
  children: null
}
```

In cases where you need the underlying element and its associated shadow boundary, you can use a standard `ref` which will be invoked with the host element &ndash; from that you can use `shadowRoot` to access its shadow root if the `mode` has been set to the default `open` value.

```javascript
const node = useRef(null)

// ...

;<root.section ref={node} />
```

Recently and at long last there has been some movement in introducing a [declarative shadow DOM](https://tomalec.github.io/declarative-shadow-dom/) which `react-shadow` _tentatively_ supports &ndash; as it's experimental, open to sudden spec changes, and React finds it difficult to rehydrate &ndash; by using the `ssr` prop.

```javascript
const node = useRef(null)

// ...

;<root.section ssr />
```
