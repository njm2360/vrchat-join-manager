import { useMemo, useRef, useState } from 'react'
import { Box, Button, Stack } from '@mui/material'
import { DateTimePicker } from '@mui/x-date-pickers/DateTimePicker'
import type { Dayjs } from 'dayjs'
import { Line } from 'react-chartjs-2'
import type { ChartOptions, ChartData } from 'chart.js'
import type { Chart } from 'chart.js'
import { useTimeline } from '../../api/queries'
import type { InstanceOut } from '../../api/schemas'
import { chartZoomOptions, visibleYRangePlugin } from '../../utils/chart'

interface Props {
  instanceId: number
  instance: InstanceOut | null
  onCompare: () => void
}

type Pt = { x: Date; y: number; displayName?: string | null }

export default function TimelineTab({ instanceId, instance, onCompare }: Props) {
  const [start, setStart] = useState<Dayjs | null>(null)
  const [end, setEnd] = useState<Dayjs | null>(null)
  const [applied, setApplied] = useState<{ start?: string; end?: string }>({})
  const chartRef = useRef<Chart<'line'> | null>(null)

  const { data: timeline = [] } = useTimeline(instanceId, applied)

  const isOngoing = instance && !instance.closed_at
  const points: Pt[] = useMemo(() => {
    const pts: Pt[] = timeline.map((d) => ({
      x: new Date(d.timestamp),
      y: d.count,
      displayName: d.display_name,
    }))
    if (isOngoing && pts.length > 0) {
      pts.push({ x: new Date(), y: pts[pts.length - 1].y, displayName: null })
    }
    return pts
  }, [timeline, isOngoing])

  const data: ChartData<'line', Pt[]> = {
    datasets: [
      {
        label: '人数',
        data: points,
        borderColor: 'rgb(13, 110, 253)',
        backgroundColor: 'rgba(13, 110, 253, 0.08)',
        stepped: true,
        fill: true,
        pointRadius: points.length < 200 ? 3 : 0,
        borderWidth: 2,
      },
    ],
  }

  const options: ChartOptions<'line'> = {
    responsive: true,
    maintainAspectRatio: false,
    scales: {
      x: {
        type: 'time',
        time: { displayFormats: { minute: 'HH:mm', hour: 'MM/dd HH:mm' } },
        max: isOngoing ? Date.now() : undefined,
      },
      y: {
        beginAtZero: true,
        ticks: { stepSize: 1 },
        title: { display: true, text: '人数' },
      },
    },
    plugins: {
      legend: { display: false },
      tooltip: {
        callbacks: {
          title: (items) =>
            new Date(items[0].parsed.x ?? 0).toLocaleString('ja-JP'),
          label: (item) => ` ${item.parsed.y} 人`,
          afterLabel: (item) => {
            const raw = item.dataset.data[item.dataIndex] as unknown as Pt
            return raw?.displayName ? ` ${raw.displayName}` : ''
          },
        },
      },
      zoom: chartZoomOptions,
    },
  }

  return (
    <Stack spacing={2}>
      <Stack direction="row" spacing={1} useFlexGap sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
        <DateTimePicker
          label="開始"
          value={start}
          onChange={setStart}
          slotProps={{ textField: { size: 'small' } }}
        />
        <Box className="text-neutral-500 text-sm">〜</Box>
        <DateTimePicker
          label="終了"
          value={end}
          onChange={setEnd}
          slotProps={{ textField: { size: 'small' } }}
        />
        <Button
          variant="contained"
          size="small"
          onClick={() =>
            setApplied({
              start: start?.toISOString(),
              end: end?.toISOString(),
            })
          }
        >
          更新
        </Button>
        <Button
          variant="outlined"
          size="small"
          onClick={() => chartRef.current?.resetZoom()}
        >
          ズームリセット
        </Button>
        <Box className="ml-auto">
          <Button variant="outlined" size="small" onClick={onCompare}>
            他のインスタンスと比較
          </Button>
        </Box>
      </Stack>
      <Box className="relative h-[420px]">
        <Line
          ref={(c) => {
            chartRef.current = c as unknown as Chart<'line'> | null
          }}
          data={data}
          options={options}
          plugins={[visibleYRangePlugin]}
        />
      </Box>
    </Stack>
  )
}
