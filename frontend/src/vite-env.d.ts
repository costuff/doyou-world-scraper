/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_ANALYTICS_API_URL: string;
  // Add more env variables here as needed
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
