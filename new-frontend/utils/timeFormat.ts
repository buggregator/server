const padTo2Digits = (num: number) => num.toString().padStart(2, '0')

export default (date: Date) => (
    [
      padTo2Digits(date.getHours()),
      padTo2Digits(date.getMinutes()),
      padTo2Digits(date.getSeconds()),
    ].join(':')
  )
