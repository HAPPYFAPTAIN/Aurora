import { useTranslation } from 'react-i18next'
import { Plus, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import type { TTSAPIProfileSettings } from './types'

const INHERIT_VALUE = '__inherit__'
const TTS_FORMATS = ['mp3', 'opus', 'aac', 'flac', 'wav', 'pcm']

interface Props {
  profiles: TTSAPIProfileSettings[]
  effectiveProfiles: TTSAPIProfileSettings[]
  defaultProfileID: string
  effectiveDefaultProfileID: string
  onDefaultProfileChange: (id: string) => void
  onChange: (profiles: TTSAPIProfileSettings[]) => void
}

export function TTSAPIProfilesEditor({ profiles, effectiveProfiles, defaultProfileID, onDefaultProfileChange, onChange }: Props) {
  const { t } = useTranslation()

  const addProfile = () => {
    const newProfile: TTSAPIProfileSettings = {
      id: `tts-${Date.now()}`,
      name: '新 TTS Profile',
      provider: 'openai',
      openai_model: '',
      default_voice: '',
      default_format: 'mp3',
    }
    onChange([...profiles, newProfile])
    onDefaultProfileChange(newProfile.id!)
  }

  const updateProfile = (index: number, field: keyof TTSAPIProfileSettings, value: string) => {
    const updated = [...profiles]
    updated[index] = { ...updated[index], [field]: value }
    onChange(updated)
  }

  const removeProfile = (index: number) => {
    const updated = profiles.filter((_, i) => i !== index)
    onChange(updated)
    if (defaultProfileID === profiles[index].id && updated.length > 0) {
      onDefaultProfileChange(updated[0].id || '')
    }
  }

  return (
    <div className="space-y-4">
      <p className="text-xs text-[var(--nova-text-muted)]">{t('settings.ttsApi.profilesHint')}</p>

      <div className="space-y-2">
        <label className="text-xs font-medium text-[var(--nova-text-faint)]">{t('settings.ttsApi.defaultProfile')}</label>
        <Select value={defaultProfileID || INHERIT_VALUE} onValueChange={(v) => onDefaultProfileChange(v === INHERIT_VALUE ? '' : v)}>
          <SelectTrigger className="h-8">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={INHERIT_VALUE}>— {t('settings.ttsApi.defaultProfile')} —</SelectItem>
            {effectiveProfiles.map((p) => (
              <SelectItem key={p.id || p.openai_model} value={p.id || p.openai_model || ''}>
                {p.name || p.id || p.openai_model || 'unknown'}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {profiles.map((profile, index) => (
        <div key={profile.id || index} className="space-y-3 rounded-lg border border-[var(--nova-border)] p-3">
          <div className="flex items-center justify-between">
            <Input
              className="h-6 flex-1 text-xs"
              placeholder={t('settings.ttsApi.profileName')}
              value={profile.name || ''}
              onChange={(e) => updateProfile(index, 'name', e.target.value)}
            />
            <Button variant="ghost" size="icon" className="ml-2 h-5 w-5 shrink-0" onClick={() => removeProfile(index)}>
              <Trash2 className="h-3 w-3" />
            </Button>
          </div>
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="mb-1 block text-[10px] text-[var(--nova-text-faint)]">{t('settings.ttsApi.apiKey')}</label>
              <Input
                type="password"
                className="h-7 text-xs"
                placeholder={effectiveProfiles[index]?.openai_api_key ? '••••••••' : 'sk-...'}
                value={profile.openai_api_key || ''}
                onChange={(e) => updateProfile(index, 'openai_api_key', e.target.value)}
              />
            </div>
            <div>
              <label className="mb-1 block text-[10px] text-[var(--nova-text-faint)]">{t('settings.ttsApi.baseUrl')}</label>
              <Input
                className="h-7 text-xs"
                placeholder="https://api.openai.com/v1"
                value={profile.openai_base_url || ''}
                onChange={(e) => updateProfile(index, 'openai_base_url', e.target.value)}
              />
            </div>
            <div>
              <label className="mb-1 block text-[10px] text-[var(--nova-text-faint)]">{t('settings.ttsApi.model')}</label>
              <Input
                className="h-7 text-xs"
                placeholder="gpt-4o-mini-tts"
                value={profile.openai_model || ''}
                onChange={(e) => updateProfile(index, 'openai_model', e.target.value)}
              />
            </div>
            <div>
              <label className="mb-1 block text-[10px] text-[var(--nova-text-faint)]">{t('settings.ttsApi.voice')}</label>
              <Input
                className="h-7 text-xs"
                placeholder={t('settings.ttsApi.voicePlaceholder')}
                value={profile.default_voice || ''}
                onChange={(e) => updateProfile(index, 'default_voice', e.target.value)}
              />
            </div>
            <div>
              <label className="mb-1 block text-[10px] text-[var(--nova-text-faint)]">{t('settings.ttsApi.format')}</label>
              <Select value={profile.default_format || 'mp3'} onValueChange={(v) => updateProfile(index, 'default_format', v)}>
                <SelectTrigger className="h-7 text-xs">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {TTS_FORMATS.map((f) => (
                    <SelectItem key={f} value={f}>{f}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div>
              <label className="mb-1 block text-[10px] text-[var(--nova-text-faint)]">{t('settings.ttsApi.speed')}</label>
              <Input
                className="h-7 text-xs"
                placeholder="1.0"
                value={profile.default_speed || ''}
                onChange={(e) => updateProfile(index, 'default_speed', e.target.value)}
              />
            </div>
          </div>
        </div>
      ))}

      <Button variant="outline" size="sm" onClick={addProfile}>
        <Plus className="mr-1 h-3 w-3" />
        {t('settings.ttsApi.addProfile')}
      </Button>
    </div>
  )
}
