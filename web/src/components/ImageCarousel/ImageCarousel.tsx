import React, { useState } from 'react'
import { ButtonGroup, ButtonVariation, Button, Container, Dialog, Carousel } from '@harness/uicore'
import { ZOOM_INC_DEC_LEVEL } from 'utils/Utils'
import { useStrings } from 'framework/strings'
import css from './ImageCarousel.module.scss'

interface ImageCarouselProps {
  isOpen: boolean
  setIsOpen: (value: boolean) => void
  setZoomLevel: (value: number) => void
  zoomLevel: number
  imgEvent: string[]
}

const ImageCarousel = (props: ImageCarouselProps) => {
  const { getString } = useStrings()
  const { isOpen, setIsOpen, setZoomLevel, zoomLevel, imgEvent } = props
  const [imgTitle, setImageTitle] = useState(imgEvent[0])

  return (
    <Dialog
      portalClassName={css.portalContainer}
      title={imgTitle ? imgTitle.substring(imgTitle.lastIndexOf('/') + 1, imgTitle.length) : 'image'}
      autoFocus={true}
      className={css.imageModal}
      isOpen={isOpen}
      onClose={() => {
        setIsOpen(false)
        setZoomLevel(1)
      }}>
      <Container className={css.content}>
        <Carousel
          className={css.content}
          hideSlideChangeButtons={false}
          hideIndicators={false}
          onChange={index => {
            setImageTitle(imgEvent[index - 1])
          }}>
          {imgEvent &&
            imgEvent.map(image => {
              return (
                <>
                  <img
                    style={{ transform: `scale(${zoomLevel || 1})`, height: `${window.innerHeight - 200}px` }}
                    className={css.image}
                    src={image}
                  />
                </>
              )
            })}
        </Carousel>
      </Container>

      <Container className={css.vertical}>
        <ButtonGroup>
          <Button
            variation={ButtonVariation.TERTIARY}
            icon="zoom-in"
            data-testid="zoomInButton"
            tooltip={getString('zoomIn')}
            onClick={() => {
              Number(zoomLevel.toFixed(1)) < 2 && setZoomLevel(zoomLevel + ZOOM_INC_DEC_LEVEL)
            }}
          />
          {/* <Button
              variation={ButtonVariation.TERTIARY}
              icon="canvas-selector"
              data-testid="canvasButton"
              tooltip={getString('resetZoom')}
              onClick={() => setZoomLevel(INITIAL_ZOOM_LEVEL)}
            /> */}
          <Button
            variation={ButtonVariation.TERTIARY}
            icon="zoom-out"
            data-testid="zoomOutButton"
            tooltip={getString('zoomOut')}
            onClick={() => {
              Number(zoomLevel.toFixed(1)) > 0.3 && setZoomLevel(zoomLevel - ZOOM_INC_DEC_LEVEL)
            }}
          />
        </ButtonGroup>
      </Container>
    </Dialog>
  )
}

export default ImageCarousel
