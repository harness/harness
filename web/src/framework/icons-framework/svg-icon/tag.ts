/* eslint-disable @typescript-eslint/no-namespace */
import { getIcon } from './registry'

export const TAG = 'svg-icon'

const NAME = 'name'
const COLOR = 'color'
const SIZE = 'size'
const STROKE_WIDTH = 'stroke-width'
const TITLE = 'title'

let defaultColor: string
let defaultSize: string
let defaultStrokeWidth: string

export function setTagDefaultAttributes(props: Omit<TagProps, 'name' | 'title'> = {}) {
  defaultColor = props.color || 'currentColor'
  defaultSize = props.size || '16px'
  defaultStrokeWidth = props[STROKE_WIDTH] || '1'
}

setTagDefaultAttributes()

if (!customElements.get(TAG)) {
  customElements.define(
    TAG,
    class extends HTMLElement {
      #shadowRoot: ShadowRoot
      #connected = false
      #attrs: Partial<TagProps> = {}
      #svg: Element | undefined

      constructor() {
        super()
        this.#shadowRoot = this.attachShadow({ mode: 'open' })
      }

      static get observedAttributes() {
        return [NAME, COLOR, SIZE, STROKE_WIDTH, TITLE]
      }

      attributeChangedCallback(attr: string, _prevValue: string, nextValue: string) {
        if (nextValue) {
          this.#attrs[attr as keyof TagProps] = nextValue
        }

        if (this.#connected) {
          if (attr === NAME) {
            this.fetchIconFromName()
          }
          this.render()
        }
      }

      fetchIconFromName() {
        const { name = '' } = this.#attrs
        const rawSVG = getIcon(name)

        this.setAttribute('aria-label', name.split('/')[0])
        this.setAttribute('role', 'img')

        if (rawSVG) {
          const fragment = document.createRange().createContextualFragment(rawSVG)
          const svg = fragment.firstElementChild as Element

          if (this.#svg) {
            this.#shadowRoot.removeChild(this.#svg)
          }

          this.#shadowRoot.appendChild(svg)
          this.#svg = svg
        } else {
          // eslint-disable-next-line no-console
          console.error('[svg-icon]: Icon not found:', name)
        }
      }

      connectedCallback() {
        this.fetchIconFromName()
        this.render()
        this.#connected = true
      }

      disconnectedCallback() {
        this.#connected = false
      }

      render() {
        const {
          size = defaultSize,
          color = defaultColor,
          [STROKE_WIDTH]: strokeWidth = defaultStrokeWidth
        } = this.#attrs
        let [width, height] = size.split('/')

        // Convert to px if width/height represent a number
        // Set height = width if width is not passed
        width = +width === +width ? `${width}px` : width
        height = height ? (+height === +height ? `${height}px` : height) : width

        const styleSheet = new CSSStyleSheet()
        styleSheet.replaceSync(
          `:host { all: initial; display: inline-block; width: ${width}; height: ${height}; color: ${color}; flex-shrink: 0; cursor: inherit; } svg { width: 100%; height: 100%; stroke-width: ${strokeWidth}; }`
        )
        this.#shadowRoot.adoptedStyleSheets = [styleSheet]

        if (this.#svg) {
          this.#svg.removeAttribute('width')
          this.#svg.removeAttribute('height')
          this.#svg.removeAttribute(STROKE_WIDTH)
        }
      }
    }
  )
}

export interface TagProps {
  class?: string

  /**
   * Icon name. During generation, icon name is postfixed with icon-set name.
   * For example: code/harness, arrow-left/noir.
   */
  name: string

  /**
   * Icon size. Size can be passed under multiple formats:
   * Value           | Meaning
   *  "16"           |  16px width x 16px height
   *  "16px"         |  16px width x 16px height
   *  "2em"          |  2em width x 2em height
   *  "100/120"      |  100px width x 200px height
   *  "100px/120px"  |  100px width x 200px height
   */
  size?: string

  /**
   * Icon color.
   */
  color?: string

  /**
   * SVG stroke width.
   */
  'stroke-width'?: string

  /**
   * Title attribute.
   */
  title?: string
}

declare global {
  namespace JSX {
    interface IntrinsicElements {
      'svg-icon': TagProps
    }
  }
}
