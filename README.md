# FSSAI License Navigator

A free, single-binary tool that works out which of the three FSSAI tiers a food business needs — **Basic Registration**, **State Licence** or **Central Licence** — tells you exactly which rule put you there, lists the documents to carry, prices the indicative government fee, and watches every licence expiry date before it lapses.

It runs on your own machine, serves a small web interface from the binary itself, and stores everything in one local JSON file.

## What it does

Answer a short questionnaire — annual turnover, what the business does, food made or processed per day, milk handled per day, how many states you operate in, and whether you import, export, run a 100% export oriented unit, sell through e-commerce or occupy premises inside an airport or seaport. The tool returns:

- **the tier**, with the form to file it in (Form A for a registration, Form B for a licence) and the authority that issues it;
- **the reasoning chain** — every threshold that fired and every one that cleared, in the order the rules were applied, quoting the figure each answer was compared against;
- **the document checklist** for that tier and that kind of business, with who each document is asked of and what satisfies it, exportable as CSV;
- **the indicative government fee**, priced for each validity from one to five years;
- **a renewal watchdog** over the licences you save, sorted by urgency, with alerts at 90, 30 and 7 days, an overdue state after that, and the indicative late fee accrued so far.

Being told "State Licence" is not useful on its own. Being told *"milk handled 3,000 litres a day falls in the 500 to 50,000 litres a day band, which is a State Licence"* is something you can check and act on. That explanation is the point of the tool.

## Who it's for

Food manufacturers, processors, traders, wholesalers, retailers, restaurants, caterers, transporters, cold stores, home kitchens and cloud kitchens in **India** — the small operators without a consultant on retainer, who most often apply for the wrong tier or let a licence lapse because nobody was watching the date. It is just as useful if you already hold a licence: growing turnover, a second state or a first export can move you into a different tier underneath you.

## The regulation

The three tiers come from the **Food Safety and Standards Act, 2006** and the **Food Safety and Standards (Licensing and Registration of Food Businesses) Regulations, 2011**. Applications are filed on **FoSCoS**, the Food Safety Compliance System operated by the Food Safety and Standards Authority of India.

The rules are applied in this order:

1. **Central Licence** if any one of these is true — turnover above ₹20 crore; imports food; exports food or is a 100% export oriented unit; is an e-commerce food business operator; operates in two or more states counted from the head office; premises inside an airport or seaport; or production capacity above what a State Licence covers.
2. **Basic Registration** if none of those apply and the business is a petty food business operator — turnover at or below ₹12 lakh, food capacity at or below 100 kg a day, and milk at or below 500 litres a day.
3. **State Licence** for everything in between.

The capacity ceilings separating a State Licence from a Central Licence are 2 metric tonnes a day of food manufacturing or processing, and 50,000 litres a day of milk, or 2,500 metric tonnes of milk solids a year. Milk between 500 and 50,000 litres a day sits in the State Licence band.

## Install and run

One binary, no dependencies outside the Go standard library. With Go 1.25 or later:

```sh
# build it once, then run the binary anywhere
go build -o fssai-license-navigator .
./fssai-license-navigator

# or run it straight from the source directory
go run .

# choose a different port or data file
./fssai-license-navigator -addr 127.0.0.1:8137 -data ~/licences.json
```

Then open <http://127.0.0.1:8080/>.

| Flag | Default | Meaning |
| --- | --- | --- |
| `-addr` | `127.0.0.1:8080` | Address to listen on. Loopback only, so the tool is not reachable from your network. |
| `-data` | `./fssai-license-navigator-data.json` | Path to the local data file. |

Run the tests with `go test ./...`.

## Where your data is stored

In the single JSON file named by `-data`, holding the licences you saved and any threshold or fee you corrected. The path is printed when the tool starts and shown in the interface. Back it up by copying that one file.

Nothing else leaves your machine. The interface is embedded in the binary, the tool binds to the loopback address, and it makes **no outbound network calls** — no account, no telemetry, no analytics, no third-party scripts or fonts. The data file is written to a temporary file and renamed into place, so an interrupted write cannot corrupt what you already had.

## Indicative figures

Every statutory figure is a named constant in one place (`internal/fssai/params.go`), is stated in the interface, and can be corrected inside the tool if the authority revises it — the decision then follows your figure.

| Figure | Value used |
| --- | --- |
| Petty food business operator turnover ceiling | ₹12,00,000 a year |
| Turnover above which a Central Licence is needed | ₹20,00,00,000 a year |
| Petty food business operator food capacity | 100 kg a day |
| Food capacity a State Licence covers | 2 metric tonnes a day |
| Petty food business operator milk ceiling | 500 litres a day |
| Milk a State Licence covers | 50,000 litres a day |
| Milk solids a State Licence covers | 2,500 metric tonnes a year |
| States of operation triggering a Central Licence | 2 |
| Basic Registration fee | ₹100 a year |
| State Licence fee | ₹2,000 a year (food service, and any other food business); ₹3,000 a year (manufacturing or processing at or below 1 MT/day of food and 10,000 litres/day of milk); ₹5,000 a year above that band |
| Central Licence fee | ₹7,500 a year |
| Validity | 1 to 5 years, chosen at application; the fee is per year |
| Renewal alerts | 90, 30 and 7 days before expiry |
| Indicative late renewal penalty | ₹100 a day after expiry |

A licence counts as valid through the whole of its expiry date, so the day of expiry is nought days remaining and not yet overdue.

## Scope and accuracy

**These figures are indicative and they change.** FSSAI revises fees, turnover thresholds and capacity ceilings from time to time, and the required documents vary by state, by food category and by the kind of entity you are. Production capacity and milk volumes are read exactly as you declare them, so traders, retailers and food service operators should leave them at zero; the milk solids figure only decides the step up to a Central Licence, because the petty ceiling is written in litres a day.

This is a planning aid. It is not legal advice, it is not a filing service, and it does not submit anything to FSSAI on your behalf. Confirm your eligibility, the current fee and the required documents on the official FoSCoS portal, or with your state's Food Safety Commissionerate, before you apply or renew.

MIT licensed.
