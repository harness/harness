import type * as Diff2Html from 'diff2html'
import HoganJsUtils from 'diff2html/lib/hoganjs-utils'
import 'highlight.js/styles/github.css'
import 'diff2html/bundles/css/diff2html.min.css'

export enum ViewStyle {
  SIDE_BY_SIDE = 'side-by-side',
  LINE_BY_LINE = 'line-by-line'
}

export const DIFF_VIEWER_HEADER_HEIGHT = 36
export const INITIAL_ANNOTATION_HEIGHT = 214

export const DIFF2HTML_CONFIG = {
  outputFormat: 'side-by-side',
  drawFileList: false,
  fileListStartVisible: false,
  fileContentToggle: true,
  matching: 'lines',
  synchronisedScroll: true,
  highlight: true,
  renderNothingWhenEmpty: false,
  compiledTemplates: {
    'generic-line': HoganJsUtils.compile(`
      <tr>
        <td class="{{lineClass}} {{type}}">
          {{{lineNumber}}}
        </td>
        <td class="{{type}}" data-content-for-line-number="{{lineNumber}}">
            <div data-annotation-for-line="{{lineNumber}}" tab-index="0" role="button">+</div>
            <div class="{{contentClass}}">
            {{#prefix}}
                <span class="d2h-code-line-prefix">{{{prefix}}}</span>
            {{/prefix}}
            {{^prefix}}
                <span class="d2h-code-line-prefix">&nbsp;</span>
            {{/prefix}}
            {{#content}}
                <span class="d2h-code-line-ctn">{{{content}}}</span>
            {{/content}}
            {{^content}}
                <span class="d2h-code-line-ctn"><br></span>
            {{/content}}
            </div>
        </td>
      </tr>
    `),
    'side-by-side-file-diff': HoganJsUtils.compile(`
      <div id="{{fileHtmlId}}" class="d2h-file-wrapper side-by-side-file-diff" data-lang="{{file.language}}">
        <div class="d2h-file-header">
          {{{filePath}}}
        </div>
        <div class="d2h-files-diff">
            <div class="d2h-file-side-diff left">
                <div class="d2h-code-wrapper">
                    <table class="d2h-diff-table">
                        <tbody class="d2h-diff-tbody">
                        {{{diffs.left}}}
                        </tbody>
                    </table>
                </div>
            </div>
            <div class="d2h-file-side-diff right">
                <div class="d2h-code-wrapper">
                    <table class="d2h-diff-table">
                        <tbody class="d2h-diff-tbody">
                        {{{diffs.right}}}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
      </div>
    `),
    'line-by-line-file-diff': HoganJsUtils.compile(`
      <div id="{{fileHtmlId}}" class="d2h-file-wrapper line-by-line-file-diff" data-lang="{{file.language}}">
        <div class="d2h-file-header">
        {{{filePath}}}
        </div>
        <div class="d2h-file-diff">
            <div class="d2h-code-wrapper">
                <table class="d2h-diff-table">
                    <tbody class="d2h-diff-tbody">
                    {{{diffs}}}
                    </tbody>
                </table>
            </div>
        </div>
      </div>
    `),
    'line-by-line-numbers': HoganJsUtils.compile(`
      <div class="line-num1">{{oldNumber}}</div>
      <div class="line-num2">{{newNumber}}</div>
    `)
  }
} as Readonly<Diff2Html.Diff2HtmlConfig>

// export function buildSideBySide
