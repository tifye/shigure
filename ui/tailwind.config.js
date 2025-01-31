/** @type {import('tailwindcss').Config} */
module.exports = {
    content: ["./src/**/*.{html,ts,tsx}"],
    theme: {
        extend: {
            fontFamily: {
                fira: ["Fira Mono", "monospace"],
                manrope: ["Manrope", "sans-serif"],
                konigsberg: ["Konigsberg", "sans-serif"],
            },
            screens: {
                "3xl": "1750px",
                desktop: "1920px",
                wide: "2200px",
                "2k": "2500px",
            },
        },
    },
}
