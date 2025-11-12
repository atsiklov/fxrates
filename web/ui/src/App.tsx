import { useEffect, useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import './App.css'
import { fetchLatestRate, fetchRateUpdate, fetchSupportedCurrencies, scheduleRateUpdate } from './api'
import type { RateUpdateView } from './api'
import { ArrowPathIcon } from '@heroicons/react/24/outline'

type UpdateRow = RateUpdateView & {
  requestedAt: string
  checking?: boolean
  error?: string | null
}

type LatestRateState = {
  base: string
  quote: string
  value: number
  updatedAt: string
}

const formatDate = (value?: string) => {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.valueOf())) return value
  return date.toLocaleString()
}

function App() {
  const [codes, setCodes] = useState<string[]>([])
  const [baseCode, setBaseCode] = useState('')
  const [quoteCode, setQuoteCode] = useState('')
  const [codesError, setCodesError] = useState<string | null>(null)
  const [isCodesLoading, setIsCodesLoading] = useState(true)

  const [latestRate, setLatestRate] = useState<LatestRateState | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [isRateLoading, setIsRateLoading] = useState(false)

  const [updates, setUpdates] = useState<UpdateRow[]>([])
  const [isScheduling, setIsScheduling] = useState(false)
  const [pendingPopupVisible, setPendingPopupVisible] = useState(false)
  const [checkPopupId, setCheckPopupId] = useState<string | null>(null)

  useEffect(() => {
    if (!checkPopupId) {
      return
    }
    const timer = window.setTimeout(() => setCheckPopupId(null), 400)
    return () => window.clearTimeout(timer)
  }, [checkPopupId])

  useEffect(() => {
    if (!pendingPopupVisible) {
      return
    }
    const timer = window.setTimeout(() => setPendingPopupVisible(false), 800)
    return () => window.clearTimeout(timer)
  }, [pendingPopupVisible])

  useEffect(() => {
    const load = async () => {
      try {
        setIsCodesLoading(true)
      const fetchedCodes = (await fetchSupportedCurrencies()) ?? []
        setCodes(fetchedCodes)
        setBaseCode((prev) => prev || fetchedCodes[0] || '')
        setQuoteCode((prev) => prev || fetchedCodes[1] || fetchedCodes[0] || '')
        setCodesError(null)
      } catch (err) {
        setCodesError(err instanceof Error ? err.message : 'Failed to load currencies')
      } finally {
        setIsCodesLoading(false)
      }
    }
    load()
  }, [])

  const disableActions = useMemo(() => {
    if (!baseCode || !quoteCode) return true
    if (isCodesLoading) return true
    return false
  }, [baseCode, quoteCode, isCodesLoading])

  const handleSelectChange = (setter: (value: string) => void) => (event: React.ChangeEvent<HTMLSelectElement>) => {
    setter(event.target.value)
  }

  const handleGetLatest = async () => {
    if (disableActions) return
    setIsRateLoading(true)
    setActionError(null)
    try {
      const rate = await fetchLatestRate(baseCode, quoteCode)
      setLatestRate(rate)
    } catch (err) {
      setLatestRate(null)
      setActionError(err instanceof Error ? err.message : 'Failed to load rate')
    } finally {
      setIsRateLoading(false)
    }
  }

  const handleScheduleUpdate = async (event: FormEvent) => {
    event.preventDefault()
    if (disableActions) return

    setIsScheduling(true)
    setActionError(null)
    try {
      const { updateId } = await scheduleRateUpdate(baseCode, quoteCode)
      const existedBefore = updates.some((item) => item.updateId === updateId)
      const latest = await fetchRateUpdate(updateId)

      setUpdates((prev) => {
        if (existedBefore) {
          return prev.map((item) =>
            item.updateId === updateId
              ? {
                  ...item,
                  status: latest.status,
                  value: latest.value ?? item.value,
                  updatedAt: latest.updatedAt ?? item.updatedAt,
                }
              : item,
          )
        }
        const nextRow: UpdateRow = {
          updateId,
          base: latest.base || baseCode,
          quote: latest.quote || quoteCode,
          status: latest.status,
          requestedAt: new Date().toISOString(),
          value: latest.value,
          updatedAt: latest.updatedAt,
        }
        return [nextRow, ...prev]
      })

      if (existedBefore && latest.status === 'pending') {
        setPendingPopupVisible(true)
        return
      }
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to schedule update')
    } finally {
      setIsScheduling(false)
    }
  }

  const handleCheckUpdate = async (row: UpdateRow) => {
    setUpdates((prev) =>
      prev.map((item) => (item.updateId === row.updateId ? { ...item, checking: true, error: null } : item)),
    )
    try {
      const latest = await fetchRateUpdate(row.updateId)
      setUpdates((prev) =>
        prev.map((item) =>
          item.updateId === row.updateId
            ? {
                ...item,
                status: latest.status,
                value: latest.value ?? item.value,
                updatedAt: latest.updatedAt ?? item.updatedAt,
                checking: false,
                error: null,
              }
            : item,
        ),
      )
      if (latest.status === 'pending') {
        setCheckPopupId(row.updateId)
      }
    } catch (err) {
      setUpdates((prev) =>
        prev.map((item) =>
          item.updateId === row.updateId
            ? {
                ...item,
                checking: false,
                error: err instanceof Error ? err.message : 'Failed to check update status',
              }
            : item,
        ),
      )
    }
  }

  return (
    <main className="app">
      <h1>FX Rates</h1>
      <form className="controls" onSubmit={handleScheduleUpdate}>
        <div className="control-row">
          <label>
            Base currency
            <select value={baseCode} onChange={handleSelectChange(setBaseCode)} disabled={isCodesLoading || !codes.length}>
              {codes.map((code) => (
                <option value={code} key={code}>
                  {code}
                </option>
              ))}
            </select>
          </label>
          <label>
            Quote currency
            <select value={quoteCode} onChange={handleSelectChange(setQuoteCode)} disabled={isCodesLoading || !codes.length}>
              {codes.map((code) => (
                <option value={code} key={code}>
                  {code}
                </option>
              ))}
            </select>
          </label>

          <div className="buttons">
            <button
              type="button"
              className="btn-secondary action-btn"
              onClick={handleGetLatest}
              disabled={disableActions || isRateLoading}
            >
              {isRateLoading ? 'Loading...' : 'Get latest rate'}
            </button>
            <button type="submit" className="btn-primary action-btn" disabled={disableActions || isScheduling}>
              {isScheduling ? 'Scheduling...' : 'Update rate'}
            </button>
            {pendingPopupVisible && <div className="button-popup">Still pending</div>}
          </div>
        </div>
      </form>

      {codesError && <p className="error">Currencies: {codesError}</p>}
      {actionError && <p className="error">{actionError}</p>}

      <section className="latest-rate">
        <header>
          <h2>Latest rate</h2>
          {!latestRate && <span className="hint">Choose a pair and click &quot;Get latest rate&quot;</span>}
        </header>
        {latestRate && (
          <dl>
            <div>
              <dt>Pair</dt>
              <dd>
                {latestRate.base}/{latestRate.quote}
              </dd>
            </div>
            <div>
              <dt>Value</dt>
              <dd className="value">{latestRate.value}</dd>
            </div>
            <div>
              <dt>Updated</dt>
              <dd>{formatDate(latestRate.updatedAt)}</dd>
            </div>
          </dl>
        )}
      </section>

      <section className="updates">
        <header>
          <h2>Scheduled updates</h2>
          <span className="hint">Use &quot;Update rate&quot; to add a new row</span>
        </header>

        {updates.length === 0 ? (
          <p className="empty">No updates yet.</p>
        ) : (
          <div className="table-wrapper">
            <table>
              <colgroup>
                <col className="col-update-id" />
                <col className="col-pair" />
                <col className="col-status" />
                <col className="col-requested" />
                <col className="col-updated" />
                <col className="col-value" />
              </colgroup>
              <thead>
                <tr>
                  <th>Update ID</th>
                  <th>Base / Quote</th>
                  <th>Status</th>
                  <th>Requested</th>
                  <th>Updated</th>
                  <th>Value</th>
                </tr>
              </thead>
              <tbody>
                {updates.map((row) => (
                  <tr key={row.updateId}>
                    <td>{row.updateId}</td>
                    <td>
                      {row.base}/{row.quote}
                    </td>
                    <td className={`status status-${row.status}`}>{row.status}</td>
                    <td>{formatDate(row.requestedAt)}</td>
                    <td>{formatDate(row.updatedAt)}</td>
                    <td className="value-cell">
                      <div className="value-main">
                        {row.status === 'pending' ? (
                          <button
                            type="button"
                            className="value-action"
                            onClick={() => handleCheckUpdate(row)}
                            disabled={row.checking}
                            aria-label="Check update status"
                          >
                            <ArrowPathIcon className={`value-icon ${row.checking ? 'spinning' : ''}`} />
                          </button>
                        ) : (
                          <span className="value-text">{row.value ?? '--'}</span>
                        )}
                        {checkPopupId === row.updateId && <div className="row-popup">Still pending</div>}
                      </div>
                      {row.error && <div className="row-error">{row.error}</div>}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </main>
  )
}

export default App



