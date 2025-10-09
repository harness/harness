import React from 'react'
import './MinimalDonutChart.module.scss'

export interface MinimalDonutChartProps {
  size: number
  innerRadius: number
  colors: string[]
  degree: number
  background: string
}

const MinimalDonutChart = ({
  size = 32,
  innerRadius = 70,
  colors = ['#ffffff', '#4CAF50'],
  degree = 0,
  background
}: MinimalDonutChartProps) => {
  const donutSize = `${size}px`
  const donutInnerSize = `${(size * innerRadius) / 100}px` // Calculate inner circle size based on percentage

  const donutStyle = {
    width: donutSize,
    height: donutSize,
    background: `conic-gradient(${colors[0]} 0deg, ${colors[0]} ${degree}deg, ${colors[1]} ${degree}deg, ${colors[1]} 360deg)`,
    position: 'relative',
    borderRadius: '50%'
  } as any

  const innerCircleStyle = {
    width: donutInnerSize,
    height: donutInnerSize,
    position: 'absolute',
    top: '50%',
    left: '50%',
    transform: 'translate(-50%, -50%)',
    backgroundColor: background,
    borderRadius: '50%'
  } as any

  return (
    <div className="donut-chart" style={donutStyle}>
      <div className="donut-inner" style={innerCircleStyle}></div>
    </div>
  )
}

export default MinimalDonutChart
