import createClient from "openapi-fetch";
import type { paths } from "@/api/types.gen";

const baseUrl = document.baseURI.replace(/\/$/, "");

export const api = createClient<paths>({ baseUrl });

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}
