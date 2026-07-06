import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { QueryClientProvider } from "@tanstack/react-query";
import { ThemeProvider } from "@mui/material/styles";
import CssBaseline from "@mui/material/CssBaseline";
import { LocalizationProvider } from "@mui/x-date-pickers";
import { AdapterDayjs } from "@mui/x-date-pickers/AdapterDayjs";
import { SnackbarProvider } from "notistack";
import "dayjs/locale/ja";

import "./index.css";
import "./chartSetup";
import { theme } from "@/theme";
import App from "@/App";
import { queryClient } from "@/api/queryClient";
import { PlayerDetailProvider } from "@/components/PlayerDetailProvider";

const basename = new URL(document.baseURI).pathname.replace(/\/$/, "");

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        <LocalizationProvider dateAdapter={AdapterDayjs} adapterLocale="ja">
          <SnackbarProvider
            maxSnack={3}
            autoHideDuration={3500}
            anchorOrigin={{ vertical: "bottom", horizontal: "right" }}
          >
            <BrowserRouter basename={basename}>
              <PlayerDetailProvider>
                <App />
              </PlayerDetailProvider>
            </BrowserRouter>
          </SnackbarProvider>
        </LocalizationProvider>
      </ThemeProvider>
    </QueryClientProvider>
  </StrictMode>,
);
