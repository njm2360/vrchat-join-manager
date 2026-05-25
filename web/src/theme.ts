import { createTheme } from '@mui/material/styles'
import { jaJP as coreJaJP } from '@mui/material/locale'
import { jaJP as gridJaJP } from '@mui/x-data-grid/locales'
import { jaJP as pickersJaJP } from '@mui/x-date-pickers/locales'

export const theme = createTheme(
  {
    palette: {
      mode: 'light',
      primary: { main: '#0d6efd' },
      error: { main: '#dc3545' },
      warning: { main: '#ffc107' },
      success: { main: '#198754' },
      background: { default: '#f8f9fa' },
    },
    typography: {
      fontFamily: [
        '"Roboto"',
        '"Hiragino Kaku Gothic ProN"',
        '"Noto Sans JP"',
        'sans-serif',
      ].join(','),
    },
    shape: { borderRadius: 6 },
  },
  coreJaJP,
  gridJaJP,
  pickersJaJP,
)
