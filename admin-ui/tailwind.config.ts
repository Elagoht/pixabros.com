import { type Config } from "tailwindcss";

const config: Config = {
	content: ["./index.html", "./src/**/*.{ts,tsx}"],
	theme: {
		extend: {
			fontFamily: {
				sans: ["Work Sans", "sans-serif"],
			},
			colors: {
				gray: {
					50: "#f8f9fb",
					100: "#f1f2f5",
					200: "#e4e5ea",
					300: "#c8cad2",
					400: "#9ea1ad",
					500: "#71747f",
					600: "#505360",
					700: "#3c3e4a",
					800: "#2f3139",
					900: "#24262d",
					950: "#16181d",
					1000: "#0f1013",
				},
				primary: {
					50: "#f2f5fc",
					100: "#e1e8f8",
					200: "#cbd8f2",
					300: "#a6bfea",
					400: "#7c9dde",
					500: "#5d7dd4",
					600: "#4963c7",
					700: "#3f51b5",
					800: "#394494",
					900: "#323c76",
					950: "#222749",
					1000: "#0a0d1a",
				},
				secondary: {
					50: "#fff7ed",
					100: "#ffedd5",
					200: "#fed7aa",
					300: "#fdba74",
					400: "#fb923c",
					500: "#f97316",
					600: "#ea580c",
					700: "#c2410c",
					800: "#9a3412",
					900: "#7c2d12",
					950: "#431407",
					1000: "#1a0700",
				},
			},
		},
	},
	plugins: [],
};

export default config;
