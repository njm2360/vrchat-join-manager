import { lazy, Suspense } from "react";
import { Routes, Route, Navigate } from "react-router-dom";
import { Box, CircularProgress } from "@mui/material";
import AppLayout from "@/components/AppLayout";

const InstancesPage = lazy(() => import("@/pages/InstancesPage"));
const ComparePage = lazy(() => import("@/pages/ComparePage"));
const PlayerSearchPage = lazy(() => import("@/pages/PlayerSearchPage"));
const PlayerPage = lazy(() => import("@/pages/PlayerPage"));

function PageFallback() {
  return (
    <Box className="h-full flex items-center justify-center">
      <CircularProgress size={32} />
    </Box>
  );
}

export default function App() {
  return (
    <Routes>
      <Route element={<AppLayout />}>
        <Route
          path="/"
          element={
            <Suspense fallback={<PageFallback />}>
              <InstancesPage />
            </Suspense>
          }
        />
        <Route
          path="/instances/:instanceId"
          element={
            <Suspense fallback={<PageFallback />}>
              <InstancesPage />
            </Suspense>
          }
        />
        <Route
          path="/violations"
          element={
            <Suspense fallback={<PageFallback />}>
              <ComparePage />
            </Suspense>
          }
        />
        <Route
          path="/players"
          element={
            <Suspense fallback={<PageFallback />}>
              <PlayerSearchPage />
            </Suspense>
          }
        />
        <Route
          path="/players/:userId"
          element={
            <Suspense fallback={<PageFallback />}>
              <PlayerPage />
            </Suspense>
          }
        />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}
