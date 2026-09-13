/* Shared seed data for every sparstrowgen prototype.
 *
 * One fake deployment: the same people, the same conversations, the same
 * invitations, so a reviewer can follow one person from registration into the
 * app they land in. Prototypes read it; nothing writes it back.
 *
 * Not real data. The owner's address in particular was not looked up from
 * production. agent@sparstrow.com is a real mailbox the owner provided for
 * testing, which is why it appears as an invited address with no account.
 */
window.SPARSTROW_SEED = {
  sender: "sparstrowgen <no-reply@sparstrow.com>",

  accounts: [
    {
      email: "srihari@sparstrow.com",
      password: "owner-password-2026",
      verified: true,
      note: "the existing owner account, created with the old setup code",
      conversations: [
        { title: "EDI 856 ASN parser rejects empty REF segments", provider: "claude", when: "12 min" },
        { title: "Stop codex hanging after turn.completed", provider: "codex", when: "1 h" },
        { title: "Coolify migrate container on the second deploy", provider: "claude", when: "3 h" },
        { title: "Invitation-only sign-up: what the email says", provider: "agy", when: "Yesterday" },
        { title: "NAV item journal posting batch size", provider: "codex", when: "Yesterday" },
        { title: "Why agy returns one chunk for short answers", provider: "agy", when: "Sep 10" },
        { title: "Folder picker walks up from D:\\sparstrowgen", provider: "claude", when: "Sep 10" },
        { title: "Capture a real claude rate_limit_event", provider: "claude", when: "Sep 9" },
        { title: "Password change signs out other sessions", provider: "codex", when: "Sep 9" },
      ],
    },
    {
      email: "marcus.bell@sparstrow.com",
      password: "warehouse-labels-26",
      verified: true,
      note: "a second person who already registered by invitation",
      conversations: [
        { title: "Zebra label printer queue drops jobs over 40", provider: "codex", when: "2 h" },
        { title: "Replace the nightly cron with pg_cron?", provider: "claude", when: "Sep 11" },
      ],
    },
  ],

  // Addresses the owner has allowed. Stored as typed; matching trims and
  // ignores case, which is why Priya's capitals are deliberate.
  invitations: [
    "srihari@sparstrow.com",
    "marcus.bell@sparstrow.com",
    "agent@sparstrow.com",
    "Priya.Nair@northwindlogistics.com",
  ],

  notInvitedExample: "jordan.reyes@contoso.com",
};
