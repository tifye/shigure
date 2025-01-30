const BASE_URL = import.meta.env.VITE_BACKEND_HOST as string

export const ActivitySVGURI = `${BASE_URL}/activity/svg`

export type ActivityData = {
    Title: string
    Author: string
    ThumbnailUrl: string
    Url: string
}

export default async function GetActivity(): Promise<ActivityData> {
    const res = await fetch(`${BASE_URL}/activity`)
    if (!res.ok) {
        throw new Error(await res.text())
    }
    return res.json();
}

