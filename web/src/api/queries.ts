import { useMutation, useQuery, useQueryClient, keepPreviousData } from "@tanstack/react-query";
import { api } from "@/api/client";
import type {
  EventOut,
  InstanceOut,
  LocationPlayerOut,
  PlayerDetailOut,
  PlayerSessionOut,
  SessionOut,
  TimelinePoint,
  VisitorOut,
} from "@/api/schemas";

type Order = "asc" | "desc";
export type SessionSortKey = "display_name" | "join_ts" | "leave_ts" | "duration_seconds";
export type PlayerSortKey = "internal_id" | "display_name" | "join_ts";
export type VisitorSortKey =
  | "display_name"
  | "first_seen"
  | "last_seen"
  | "join_count"
  | "total_duration_seconds";

export function useSetPlayerDiscord(userId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (discordId: string | null) => {
      const { error } = await api.PUT("/api/players/{user_id}/discord", {
        params: { path: { user_id: userId } },
        body: { discord_id: discordId },
      });
      if (error) throw new Error("Discord IDの更新に失敗しました");
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["player", userId] });
      // セッション系の応答にも discord_id が乗っているので合わせて無効化
      qc.invalidateQueries({ queryKey: ["players"] });
      qc.invalidateQueries({ queryKey: ["sessions"] });
    },
  });
}

export function usePlayerDetail(userId: string, options?: { enabled?: boolean }) {
  return useQuery<PlayerDetailOut>({
    queryKey: ["player", userId],
    enabled: (options?.enabled ?? true) && !!userId,
    queryFn: async () => {
      const { data, error } = await api.GET("/api/players/{user_id}", {
        params: { path: { user_id: userId } },
      });
      if (error || !data) throw new Error("failed to load player");
      return data;
    },
  });
}

export function useInstance(id: number | null, options?: { enabled?: boolean }) {
  return useQuery<InstanceOut>({
    queryKey: ["instance", id],
    enabled: id != null && (options?.enabled ?? true),
    queryFn: async () => {
      const { data, error } = await api.GET("/api/instances/{instance_id}", {
        params: { path: { instance_id: id! } },
      });
      if (error || !data) throw new Error("failed to load instance");
      return data;
    },
  });
}

export function useTimeline(id: number | null, range: { start?: string; end?: string }) {
  return useQuery<TimelinePoint[]>({
    queryKey: ["timeline", id, range.start, range.end],
    enabled: id != null,
    queryFn: async () => {
      const { data, error } = await api.GET("/api/instances/{instance_id}/presence-timeline", {
        params: {
          path: { instance_id: id! },
          query: { start: range.start, end: range.end },
        },
      });
      if (error) throw new Error("failed to load timeline");
      return data ?? [];
    },
    placeholderData: keepPreviousData,
  });
}

export function useEvents(
  id: number | null,
  params: { order: Order; start?: string; end?: string },
) {
  return useQuery<EventOut[]>({
    queryKey: ["events", id, params.order, params.start, params.end],
    enabled: id != null,
    queryFn: async () => {
      const { data, error } = await api.GET("/api/instances/{instance_id}/events", {
        params: {
          path: { instance_id: id! },
          query: { order: params.order, start: params.start, end: params.end },
        },
      });
      if (error) throw new Error("failed to load events");
      return data ?? [];
    },
    placeholderData: keepPreviousData,
  });
}

export function useSessions(
  id: number | null,
  params: { sort_by: SessionSortKey; order: Order; start?: string; end?: string },
) {
  return useQuery<SessionOut[]>({
    queryKey: ["sessions", id, params.sort_by, params.order, params.start, params.end],
    enabled: id != null,
    queryFn: async () => {
      const { data, error } = await api.GET("/api/instances/{instance_id}/sessions", {
        params: {
          path: { instance_id: id! },
          query: {
            sort_by: params.sort_by,
            order: params.order,
            start: params.start,
            end: params.end,
          },
        },
      });
      if (error) throw new Error("failed to load sessions");
      return data ?? [];
    },
    placeholderData: keepPreviousData,
  });
}

export function usePlayers(id: number | null, params: { sort_by: PlayerSortKey; order: Order }) {
  return useQuery<LocationPlayerOut[]>({
    queryKey: ["players", id, params.sort_by, params.order],
    enabled: id != null,
    queryFn: async () => {
      const { data, error } = await api.GET("/api/instances/{instance_id}/players", {
        params: {
          path: { instance_id: id! },
          query: { sort_by: params.sort_by, order: params.order },
        },
      });
      if (error) throw new Error("failed to load players");
      return data ?? [];
    },
    placeholderData: keepPreviousData,
  });
}

export function useVisitors(id: number | null, params: { sort_by: VisitorSortKey; order: Order }) {
  return useQuery<VisitorOut[]>({
    queryKey: ["visitors", id, params.sort_by, params.order],
    enabled: id != null,
    queryFn: async () => {
      const { data, error } = await api.GET("/api/instances/{instance_id}/visitors", {
        params: {
          path: { instance_id: id! },
          query: { sort_by: params.sort_by, order: params.order },
        },
      });
      if (error) throw new Error("failed to load visitors");
      return data ?? [];
    },
    placeholderData: keepPreviousData,
  });
}

export function usePlayerSessions(
  userId: string,
  params: {
    instance_id?: number;
    start?: string;
    end?: string;
    order: Order;
    limit?: number;
    world_id?: string;
  },
  options?: { enabled?: boolean },
) {
  return useQuery<PlayerSessionOut[]>({
    queryKey: [
      "player-sessions",
      userId,
      params.instance_id ?? null,
      params.start ?? null,
      params.end ?? null,
      params.order,
      params.limit ?? null,
      params.world_id ?? null,
    ],
    queryFn: async () => {
      const { data, error } = await api.GET("/api/players/{user_id}/sessions", {
        params: {
          path: { user_id: userId },
          query: {
            instance_id: params.instance_id,
            start: params.start,
            end: params.end,
            order: params.order,
            limit: params.limit,
            world_id: params.world_id,
          },
        },
      });
      if (error) throw new Error("failed to load player sessions");
      return data ?? [];
    },
    ...options,
  });
}
