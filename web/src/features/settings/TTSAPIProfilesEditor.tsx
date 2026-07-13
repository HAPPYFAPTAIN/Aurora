import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, ChevronDown, Loader2, Plus, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { fetchTTSVoices, type TTSVoice } from '@/lib/api-client'
import type { TTSAPIProfileSettings } from './types'

const INHERIT_VALUE = '__inherit__'
const TTS_FORMATS = ['mp3', 'opus', 'aac', 'flac', 'wav', 'pcm']
const TTS_PROVIDERS = [
  { value: 'openai', label: 'OpenAI 兼容' },
  { value: 'stepfun', label: '阶跃星辰 Step Fun' },
]

interface VoiceComboboxProps {
  value: string
  profileID: string | undefined
  provider: string
  placeholder: string
  onChange: (value: string) => void
}

/**
 * VoiceCombobox 是音色选择 combobox：可从下拉列表选择，也可手动输入。
 * 列表通过 GET /api/tts/voices?profile_id=xxx 拉取，失败时仅展示输入框。
 */
function VoiceCombobox({ value, profileID, provider, placeholder, onChange }: VoiceComboboxProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [voices, setVoices] = useState<TTSVoice[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(false)

  useEffect(() => {
    if (!profileID) return
    let cancelled = false
    setLoading(true)
    setError(false)
    fetchTTSVoices(profileID)
      .then((res) => {
        if (cancelled) return
        setVoices(res.voices || [])
      })
      .catch(() => {
        if (cancelled) return
        setVoices([])
        setError(true)
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => { cancelled = true }
  }, [profileID, provider])

  return (
    <div className="relative flex items-center">
      <Input
        className="h-7 flex-1 pr-7 text-xs"
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="absolute right-0 h-7 w-7 shrink-0 text-[var(--nova-text-faint)]"
            aria-label={t('settings.ttsApi.voiceSelectHint')}
          >
            {loading ? (
              <Loader2 className="h-3 w-3 animate-spin" />
            ) : (
              <ChevronDown className="h-3 w-3" />
            )}
          </Button>
        </PopoverTrigger>
        <PopoverContent
          align="end"
          sideOffset={4}
          className="w-[var(--radix-popover-trigger-width)] max-h-[min(50dvh,20rem)] overflow-y-auto rounded-[var(--nova-radius)] border border-[var(--nova-border)] bg-[var(--nova-surface-2)] p-1 text-[var(--nova-text)] shadow-[var(--nova-shadow)]"
        >
          {loading ? (
            <div className="px-2 py-2 text-xs text-[var(--nova-text-faint)]">{t('settings.ttsApi.voiceLoading')}</div>
          ) : error ? (
            <div className="px-2 py-2 text-xs text-[var(--nova-text-faint)]">{t('settings.ttsApi.voiceFetchError')}</div>
          ) : voices.length === 0 ? (
            <div className="px-2 py-2 text-xs text-[var(--nova-text-faint)]">{t('settings.ttsApi.voiceEmpty')}</div>
          ) : (
            <div role="listbox" className="space-y-0.5">
              {voices.map((voice) => {
                const selected = voice.id === value
                return (
                  <button
                    key={voice.id}
                    type="button"
                    role="option"
                    aria-selected={selected}
                    className={`flex w-full items-center gap-2 rounded-[var(--nova-radius)] px-2 py-1 text-left text-xs ${selected ? 'bg-[var(--nova-active)] text-[var(--nova-text)]' : 'text-[var(--nova-text-muted)] hover:bg-[var(--nova-hover)] hover:text-[var(--nova-text)]'}`}
                    onClick={() => {
                      onChange(voice.id)
                      setOpen(false)
                    }}
                  >
                    <span className="min-w-0 flex-1 truncate">{voice.name}</span>
                    {selected ? <Check className="h-3 w-3 shrink-0 text-[var(--nova-text-faint)]" /> : null}
                  </button>
                )
              })}
            </div>
          )}
        </PopoverContent>
      </Popover>
    </div>
  )
}

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

      {profiles.map((profile, index) => {
        const isStepFun = profile.provider === 'stepfun'
        return (
          <div key={profile.id || index} className="space-y-3 rounded-lg border border-[var(--nova-border)] p-3">
            <div className="flex items-center gap-2">
              <Input
                className="h-6 flex-1 text-xs"
                placeholder={t('settings.ttsApi.profileName')}
                value={profile.name || ''}
                onChange={(e) => updateProfile(index, 'name', e.target.value)}
              />
              <Button variant="ghost" size="icon" className="h-5 w-5 shrink-0" onClick={() => removeProfile(index)}>
                <Trash2 className="h-3 w-3" />
              </Button>
            </div>
            <div>
              <label className="mb-1 block text-[10px] text-[var(--nova-text-faint)]">{t('settings.ttsApi.provider')}</label>
              <Select value={profile.provider || 'openai'} onValueChange={(v) => updateProfile(index, 'provider', v)}>
                <SelectTrigger className="h-7 text-xs">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {TTS_PROVIDERS.map((p) => (
                    <SelectItem key={p.value} value={p.value}>{p.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
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
                  placeholder={isStepFun ? 'https://api.stepfun.com/step_plan/v1' : 'https://api.openai.com/v1'}
                  value={profile.openai_base_url || ''}
                  onChange={(e) => updateProfile(index, 'openai_base_url', e.target.value)}
                />
              </div>
              <div>
                <label className="mb-1 block text-[10px] text-[var(--nova-text-faint)]">{t('settings.ttsApi.model')}</label>
                <Input
                  className="h-7 text-xs"
                  placeholder={isStepFun ? 'stepaudio-2.5-tts' : 'gpt-4o-mini-tts'}
                  value={profile.openai_model || ''}
                  onChange={(e) => updateProfile(index, 'openai_model', e.target.value)}
                />
              </div>
              <div>
                <label className="mb-1 block text-[10px] text-[var(--nova-text-faint)]">{t('settings.ttsApi.voice')}</label>
                <VoiceCombobox
                  value={profile.default_voice || ''}
                  profileID={profile.id}
                  provider={profile.provider || 'openai'}
                  placeholder={isStepFun ? t('settings.ttsApi.voiceStepFunHint') : t('settings.ttsApi.voicePlaceholder')}
                  onChange={(v) => updateProfile(index, 'default_voice', v)}
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
            {isStepFun ? (
              <div>
                <label className="mb-1 block text-[10px] text-[var(--nova-text-faint)]">{t('settings.ttsApi.instruction')}</label>
                <Input
                  className="h-7 text-xs"
                  placeholder={t('settings.ttsApi.instructionPlaceholder')}
                  value={profile.instruction || ''}
                  onChange={(e) => updateProfile(index, 'instruction', e.target.value)}
                />
              </div>
            ) : null}
          </div>
        )
      })}

      <Button variant="outline" size="sm" onClick={addProfile}>
        <Plus className="mr-1 h-3 w-3" />
        {t('settings.ttsApi.addProfile')}
      </Button>
    </div>
  )
}
