import { createContext, useContext, useEffect, useState } from 'react'
import { getSiteSettings } from './api'

export interface SiteSettings {
  site_name: string
  site_subtitle: string
  site_notice: string
  allow_register: boolean
  register_user_group_id: number
  register_rule_limit: number
  register_expire_days: number
  theme_policy: 'classic' | 'transparent'
  background_url: string
  mobile_background_url: string
  offline_node_seconds: number
  offline_node_retention_hours: number
}

export const defaultSiteSettings: SiteSettings = {
  site_name: '转发面板', site_subtitle: '入口直出转发管理平台', site_notice: '', allow_register: false,
  register_user_group_id: 1, register_rule_limit: 100, register_expire_days: 365,
  theme_policy: 'classic', background_url: '', mobile_background_url: '', offline_node_seconds: 90,
  offline_node_retention_hours: 24,
}

const SiteContext = createContext({ settings: defaultSiteSettings, initialized: false, refresh: async () => {} })

export function SiteProvider({ children }: { children: React.ReactNode }) {
  const [settings, setSettings] = useState(defaultSiteSettings)
  const [initialized, setInitialized] = useState(false)
  const refresh = async () => {
    try {
      const res = await getSiteSettings()
      setSettings({ ...defaultSiteSettings, ...res.data })
    } catch {
      setSettings(defaultSiteSettings)
    } finally { setInitialized(true) }
  }
  useEffect(() => { refresh() }, [])
  useEffect(() => { document.title = settings.site_name }, [settings.site_name])
  return <SiteContext.Provider value={{ settings, initialized, refresh }}>{children}</SiteContext.Provider>
}

export const useSite = () => useContext(SiteContext)

export function useSiteBackground() {
  const { settings } = useSite()
  const [mobile, setMobile] = useState(window.matchMedia('(max-width: 768px)').matches)
  useEffect(() => {
    const media = window.matchMedia('(max-width: 768px)')
    const listener = () => setMobile(media.matches)
    media.addEventListener('change', listener)
    return () => media.removeEventListener('change', listener)
  }, [])
  const image = mobile ? settings.mobile_background_url || settings.background_url : settings.background_url
  return settings.theme_policy === 'transparent' && image
    ? { backgroundImage: `linear-gradient(rgba(8,15,30,.42), rgba(8,15,30,.42)), url("${image.replace(/"/g, '%22')}")`, backgroundSize: 'cover', backgroundPosition: 'center', backgroundAttachment: 'fixed' }
    : {}
}
